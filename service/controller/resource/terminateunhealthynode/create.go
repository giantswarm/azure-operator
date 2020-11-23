package terminateunhealthynode

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/badnodedetector/pkg/detector"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
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
		r.logger.LogCtx(ctx, "level", "debug", "message", "node auto repair is not enabled for this cluster, cancelling")
		return nil
	}

	var tenantClusterK8sClient client.Client
	{
		tenantClusterK8sClient, err = r.getTenantClusterClient(ctx, &cr)
		if err != nil {
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
		r.logger.LogCtx(ctx, "level", "debug", "message", "resetting tick node counters on all nodes in tenant cluster")
	}

	return nil
}

func (r *Resource) getTenantClusterClient(ctx context.Context, cluster *capiv1alpha3.Cluster) (client.Client, error) {
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

func (r *Resource) terminateNode(ctx context.Context, node corev1.Node, cluster capiv1alpha3.Cluster) error {
	if node.Labels["role"] != "worker" {
		return microerror.Maskf(unsupportedOperationError, "Termination of master nodes is not supported on Azure")
	}

	vmssClient, err := r.azureClientsFactory.GetVirtualMachineScaleSetsClient(ctx, cluster.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	// Scale VMSS up by one.
	var vmssName string
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "Retrieving AzureMachinePool CR")
		amp := capzexpv1alpha3.AzureMachinePool{}
		err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cluster.Namespace, Name: node.Labels[label.MachinePool]}, &amp)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", "Retrieved AzureMachinePool CR")

		r.logger.LogCtx(ctx, "level", "debug", "message", "Retrieving VMSS")
		vmssName = key.NodePoolVMSSName(&amp)
		vmss, err := vmssClient.Get(ctx, cluster.Name, vmssName)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", "Retrieved VMSS")

		newCapacity := *vmss.Sku.Capacity + 1
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Scaling up VMSS %q to %d replicas", vmssName, newCapacity))

		update := compute.VirtualMachineScaleSetUpdate{
			Sku: &compute.Sku{
				Capacity: &newCapacity,
			},
		}

		_, err = vmssClient.Update(ctx, cluster.Name, vmssName, update)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Scaled up VMSS %q to %d replicas", vmssName, newCapacity))
	}

	// Terminate faulty node.
	var instanceID string
	{
		instanceID, err := key.InstanceIDFromNode(node)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleting instance with ID %q from vmss %q", instanceID, vmssName))
		res, err := vmssClient.DeleteInstances(ctx, cluster.Name, vmssName, compute.VirtualMachineScaleSetVMInstanceRequiredIDs{InstanceIds: &[]string{instanceID}})
		if err != nil {
			return microerror.Mask(err)
		}
		_, err = vmssClient.DeleteInstancesResponder(res.Response())
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleted instance with ID %q from vmss %q", instanceID, vmssName))
	}

	// expose metric about node termination
	reportNodeTermination(cluster.Name, node.Name, instanceID)
	return nil
}
