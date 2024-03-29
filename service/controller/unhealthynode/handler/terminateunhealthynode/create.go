package terminateunhealthynode

import (
	"context"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/tenantcluster/v6/pkg/tenantcluster"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/apiextensions/v6/pkg/annotation"
	"github.com/giantswarm/apiextensions/v6/pkg/label"
	"github.com/giantswarm/badnodedetector/pkg/detector"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

const (
	nodeTerminationTickThreshold = 6
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error
	cr, err := key.ToCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// check for annotation enabling the node auto repair feature
	if _, ok := cr.Annotations[annotation.NodeTerminateUnhealthy]; !ok {
		r.logger.Debugf(ctx, "terminate unhealthy node is not enabled for this cluster, cancelling")
		return nil
	}

	var tenantClusterK8sClient client.Client
	{
		tenantClusterK8sClient, err = r.getTenantClusterClient(ctx, &cr)
		if tenant.IsAPINotAvailable(err) || tenantcluster.IsTimeout(err) {
			// The kubernetes API is not reachable. This usually happens when a new cluster is being created.
			// This makes the whole controller to fail and stops next handlers from being executed even if they are
			// safe to run. We don't want that to happen so we just return and we'll try again during next loop.
			r.logger.Debugf(ctx, "tenant API not available yet")
			r.logger.Debugf(ctx, "canceling resource")

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var detectorService *detector.Detector
	{
		detectorConfig := detector.Config{
			K8sClient: tenantClusterK8sClient,
			Logger:    r.logger,

			NotReadyTickThreshold: nodeTerminationTickThreshold,
		}

		detectorService, err = detector.NewDetector(detectorConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	nodesToTerminate, err := detectorService.DetectBadNodes(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(nodesToTerminate) > 0 {
		for _, n := range nodesToTerminate {
			err := r.terminateNode(ctx, n, cr)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		// reset tick counters on all nodes in cluster to have a graceful period after terminating nodes
		err := detectorService.ResetTickCounters(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "resetting tick node counters on all nodes in tenant cluster")
	}

	return nil
}

func (r *Resource) getTenantClusterClient(ctx context.Context, cluster *capi.Cluster) (client.Client, error) {
	var k8sClient k8sclient.Interface
	{
		restConfig, err := r.tenantRestConfigProvider.NewRestConfig(ctx, key.ClusterID(cluster), cluster.Spec.ControlPlaneEndpoint.Host)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		k8sClient, err = k8sclient.NewClients(k8sclient.ClientsConfig{
			Logger:     r.logger,
			RestConfig: rest.CopyConfig(restConfig),
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return k8sClient.CtrlClient(), nil
}

func (r *Resource) terminateNode(ctx context.Context, node corev1.Node, cluster capi.Cluster) error {
	if node.Labels["role"] != "worker" {
		// We can only terminate workers in azure.
		r.logger.Debugf(ctx, "Termination of master nodes is not supported on Azure")
		return nil
	}

	vmssClient, err := r.azureClientsFactory.GetVirtualMachineScaleSetsClient(ctx, cluster.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	// Scale VMSS up by one.
	var vmssName string
	{
		r.logger.Debugf(ctx, "Retrieving AzureMachinePool CR")
		amp := capzexp.AzureMachinePool{}
		err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cluster.Namespace, Name: node.Labels[label.MachinePool]}, &amp)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Retrieved AzureMachinePool CR")

		r.logger.Debugf(ctx, "Retrieving VMSS")
		vmssName = key.NodePoolVMSSName(&amp)
		vmss, err := vmssClient.Get(ctx, cluster.Name, vmssName)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Retrieved VMSS")

		newCapacity := *vmss.Sku.Capacity + 1
		r.logger.Debugf(ctx, "Scaling up VMSS %q to %d replicas", vmssName, newCapacity)

		update := compute.VirtualMachineScaleSetUpdate{
			Sku: &compute.Sku{
				Capacity: &newCapacity,
			},
		}

		_, err = vmssClient.Update(ctx, cluster.Name, vmssName, update)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "Scaled up VMSS %q to %d replicas", vmssName, newCapacity)
	}

	// Terminate faulty node.
	var instanceID string
	{
		instanceID, err := key.InstanceIDFromNode(node)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "Deleting instance with ID %q from vmss %q", instanceID, vmssName)
		res, err := vmssClient.DeleteInstances(ctx, cluster.Name, vmssName, compute.VirtualMachineScaleSetVMInstanceRequiredIDs{InstanceIds: &[]string{instanceID}})
		if err != nil {
			return microerror.Mask(err)
		}
		_, err = vmssClient.DeleteInstancesResponder(res.Response())
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Deleted instance with ID %q from vmss %q", instanceID, vmssName)
	}

	// expose metric about node termination
	reportNodeTermination(cluster.Name, node.Name, instanceID)
	return nil
}
