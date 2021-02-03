package nodepool

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/scalestrategy"
	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// The goal of scaleUpWorkerVMSSTransition is to double the desired number
// of nodes in worker VMSS in order to provide 1:1 mapping between new
// up-to-date nodes when draining and terminating old nodes.
// This will be done in subsequent reconciliation loops to avoid hitting the
// VMSS api too hard.
func (r *Resource) scaleUpWorkerVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if machinePool == nil {
		return currentState, microerror.Mask(ownerReferenceNotSet)
	}

	if !machinePool.GetDeletionTimestamp().IsZero() {
		r.Logger.Debugf(ctx, "MachinePool is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	allReady, err := vmsscheck.InstancesAreRunning(ctx, r.Logger, virtualMachineScaleSetVMsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	// Not all workers are Running in Azure, wait for next reconciliation loop.
	if !allReady {
		return currentState, nil
	}

	strategy := scalestrategy.Quick{}

	// Ensure the deployment is successful before we move on with scaling.
	currentDeployment, err := deploymentsClient.Get(ctx, key.ClusterID(&azureMachinePool), key.NodePoolDeploymentName(&azureMachinePool))
	if IsDeploymentNotFound(err) {
		// Deployment not found, we need to apply it again.
		return DeploymentUninitialized, microerror.Mask(err)
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	switch *currentDeployment.Properties.ProvisioningState {
	case "Failed", "Canceled":
		// Deployment is failed or canceled, I need to go back and re-apply it.
		r.Logger.Debugf(ctx, "Node Pool deployment is in state %s, we need to reapply it.", *currentDeployment.Properties.ProvisioningState)
		return DeploymentUninitialized, nil
	case "Succeeded":
		// Deployment is succeeded, safe to go on.
	default:
		// Deployment is still running, we need to wait for another reconciliation loop.
		r.Logger.Debugf(ctx, "Node Pool deployment is in state %s, waiting for it to be succeeded.", *currentDeployment.Properties.ProvisioningState)
		return currentState, nil
	}

	oldWorkersCount, err := r.countOldWorkers(ctx, azureMachinePool, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient)
	if tenantcluster.IsAPINotAvailableError(err) {
		r.Logger.Debugf(ctx, "tenant API not available yet")
		r.Logger.Debugf(ctx, "canceling resource")

		return currentState, nil
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	desiredWorkerCount := int64(oldWorkersCount * 2)
	r.Logger.Debugf(ctx, "The desired number of workers is: %d", desiredWorkerCount)

	currentWorkerCount, err := r.GetInstancesCount(ctx, virtualMachineScaleSetsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	r.Logger.Debugf(ctx, "The current number of workers is: %d", currentWorkerCount)

	if desiredWorkerCount > currentWorkerCount {
		// Disable cluster autoscaler for this nodepool.
		err = r.disableClusterAutoscaler(ctx, virtualMachineScaleSetsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}

		err = r.ScaleVMSS(ctx, virtualMachineScaleSetsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool), desiredWorkerCount, strategy)
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}

		r.Logger.Debugf(ctx, "scaled worker VMSS to %d nodes", desiredWorkerCount)

		// Let's stay in the current state.
		return currentState, nil
	}

	// We didn't scale up the VMSS, ready to move to next step.
	return WaitForWorkersToBecomeReady, nil
}

func (r *Resource) countOldWorkers(ctx context.Context, azureMachinePool v1alpha3.AzureMachinePool, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient) (int32, error) {
	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return -1, microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.Logger.Debugf(ctx, "Cluster is being deleted, skipping reconciling node pool")
		return -1, nil
	}

	// All workers ready, we can scale up if needed.
	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		r.Logger.Debugf(ctx, "finding all worker VMSS instances")

		allWorkerInstances, err = r.GetVMSSInstances(ctx, azureMachinePool)
		if err != nil {
			return -1, microerror.Mask(err)
		}

		r.Logger.Debugf(ctx, "found %d worker VMSS instances", len(allWorkerInstances))
	}

	resourceGroupName := key.ClusterID(&azureMachinePool)
	nodePoolVMSSName := key.NodePoolVMSSName(&azureMachinePool)
	vmss, err := virtualMachineScaleSetsClient.Get(ctx, resourceGroupName, nodePoolVMSSName)
	if err != nil {
		return -1, microerror.Mask(err)
	}

	var oldWorkersCount int32
	{
		for _, i := range allWorkerInstances {
			old, err := r.isWorkerInstanceFromPreviousRelease(ctx, cluster, azureMachinePool.Name, i, vmss)
			if err != nil {
				return -1, nil
			}

			if old {
				oldWorkersCount += 1
			}
		}
	}

	return oldWorkersCount, nil
}
