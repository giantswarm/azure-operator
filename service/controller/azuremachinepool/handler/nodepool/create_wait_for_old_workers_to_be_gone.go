package nodepool

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) waitForOldWorkersToBeGoneTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
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

	tenantClusterK8sClient, err := r.tenantClientFactory.GetClient(ctx, cluster)
	if tenantcluster.IsAPINotAvailableError(err) {
		r.Logger.Debugf(ctx, "tenant API not available yet")
		r.Logger.Debugf(ctx, "canceling resource")

		return currentState, nil
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.Logger.Debugf(ctx, "checking if there are old nodes still running")
	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		r.Logger.Debugf(ctx, "finding all worker VMSS instances")

		allWorkerInstances, err = r.GetVMSSInstances(ctx, virtualMachineScaleSetVMsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
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

	var oldWorkersCount int
	{
		for _, i := range allWorkerInstances {
			old, err := r.isWorkerInstanceFromPreviousRelease(ctx, tenantClusterK8sClient, azureMachinePool.Name, i, vmss)
			if err != nil {
				return DeploymentUninitialized, nil
			}

			if old != nil && *old {
				oldWorkersCount += 1
			}
		}
	}

	if oldWorkersCount > 0 {
		r.Logger.Debugf(ctx, "There are still %d workers from the previous release running", oldWorkersCount)
		return currentState, nil
	}

	// Enable cluster autoscaler for this nodepool.
	err = r.enableClusterAutoscaler(ctx, virtualMachineScaleSetsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	return DeploymentUninitialized, nil
}
