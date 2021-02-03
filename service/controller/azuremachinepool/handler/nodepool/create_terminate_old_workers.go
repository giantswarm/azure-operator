package nodepool

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/coreos/go-semver/semver"
	apiextensionslabels "github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) terminateOldWorkersTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.Logger.Debugf(ctx, "Cluster is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		r.Logger.Debugf(ctx, "finding all worker VMSS instances")

		allWorkerInstances, err = r.GetVMSSInstances(ctx, azureMachinePool)
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}

		r.Logger.Debugf(ctx, "found %d worker VMSS instances", len(allWorkerInstances))
	}

	resourceGroupName := key.ClusterID(&azureMachinePool)
	nodePoolVMSSName := key.NodePoolVMSSName(&azureMachinePool)
	vmss, err := virtualMachineScaleSetsClient.Get(ctx, resourceGroupName, nodePoolVMSSName)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.Logger.Debugf(ctx, "filtering instance IDs for old instances")

	var ids compute.VirtualMachineScaleSetVMInstanceRequiredIDs
	{
		var strIds []string
		for _, i := range allWorkerInstances {
			old, err := r.isWorkerInstanceFromPreviousRelease(ctx, cluster, azureMachinePool.Name, i, vmss)
			if tenantcluster.IsAPINotAvailableError(err) {
				r.Logger.Debugf(ctx, "tenant API not available yet")
				r.Logger.Debugf(ctx, "canceling resource")

				return currentState, nil
			} else if err != nil {
				return DeploymentUninitialized, nil
			}

			if old {
				strIds = append(strIds, *i.InstanceID)
			}
		}

		ids = compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
			InstanceIds: to.StringSlicePtr(strIds),
		}
	}

	r.Logger.Debugf(ctx, "filtered instance IDs for old instances")
	r.Logger.Debugf(ctx, "terminating %d old worker instances", len(*ids.InstanceIds))

	res, err := virtualMachineScaleSetsClient.DeleteInstances(ctx, resourceGroupName, nodePoolVMSSName, ids)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	_, err = virtualMachineScaleSetsClient.DeleteInstancesResponder(res.Response())
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	r.Logger.Debugf(ctx, "terminated %d old worker instances", len(*ids.InstanceIds))

	return WaitForOldWorkersToBeGone, nil
}

func (r *Resource) getK8sWorkerNodeForInstance(ctx context.Context, tenantClusterK8sClient ctrlclient.Client, nodePoolId string, instance compute.VirtualMachineScaleSetVM) (*corev1.Node, error) {
	name := key.NodePoolInstanceName(nodePoolId, *instance.InstanceID)

	nodeList := &corev1.NodeList{}
	labelSelector := ctrlclient.MatchingLabels{apiextensionslabels.MachinePool: nodePoolId}
	err := tenantClusterK8sClient.List(ctx, nodeList, labelSelector)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	nodes := nodeList.Items
	for _, n := range nodes {
		if n.GetName() == name {
			return &n, nil
		}
	}

	// Node related to this instance was not found.
	return nil, nil
}

func (r *Resource) isWorkerInstanceFromPreviousRelease(ctx context.Context, cluster *v1alpha3.Cluster, nodePoolId string, instance compute.VirtualMachineScaleSetVM, vmss compute.VirtualMachineScaleSet) (bool, error) {
	tenantClusterK8sClient, err := r.tenantClientFactory.GetClient(ctx, cluster)
	if err != nil {
		return false, microerror.Mask(err)
	}

	n, err := r.getK8sWorkerNodeForInstance(ctx, tenantClusterK8sClient, nodePoolId, instance)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if n == nil {
		// Kubernetes node related to this instance not found, we consider the node old.
		return true, nil
	}

	myVersion := semver.New(project.Version())

	v, exists := n.GetLabels()[label.OperatorVersion]
	if !exists {
		// Label does not exist, this normally happens when a new node is coming up but did not finish
		// its kubernetes bootstrap yet and thus doesn't have all the needed labels.
		// We'll ignore this node for now and wait for it to bootstrap correctly.
		return false, nil
	}

	nodeVersion := semver.New(v)
	if nodeVersion.LessThan(*myVersion) {
		return true, nil
	}

	return false, nil
}
