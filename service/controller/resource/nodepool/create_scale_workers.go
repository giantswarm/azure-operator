package nodepool

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/nodes/scalestrategy"
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
		r.Logger.LogCtx(ctx, "level", "debug", "message", "MachinePool is being deleted, skipping reconciling node pool")
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
		// Update DeploymentSucceeded Condition for this AzureMachinePool
		_ = r.UpdateDeploymentSucceededCondition(ctx, &azureMachinePool, nil)

		// Deployment not found, we need to apply it again.
		return DeploymentUninitialized, microerror.Mask(err)
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	defer func() {
		var currentProvisioningState *string
		if currentDeployment.Properties != nil && currentDeployment.Properties.ProvisioningState != nil {
			currentProvisioningState = currentDeployment.Properties.ProvisioningState
		}
		// Update DeploymentSucceeded Condition for this AzureMachinePool
		_ = r.UpdateDeploymentSucceededCondition(ctx, &azureMachinePool, currentProvisioningState)
	}()

	switch *currentDeployment.Properties.ProvisioningState {
	case "Failed", "Canceled":
		// Deployment is failed or canceled, I need to go back and re-apply it.
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Node Pool deployment is in state %s, we need to reapply it.", *currentDeployment.Properties.ProvisioningState))
		return DeploymentUninitialized, nil
	case "Succeeded":
		// Deployment is succeeded, safe to go on.
	default:
		// Deployment is still running, we need to wait for another reconciliation loop.
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Node Pool deployment is in state %s, waiting for it to be succeeded.", *currentDeployment.Properties.ProvisioningState))
		return currentState, nil
	}

	// All workers ready, we can scale up if needed.
	desiredWorkerCount := int64(*machinePool.Spec.Replicas * 2)
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The desired number of workers is: %d", desiredWorkerCount))

	currentWorkerCount, err := r.GetInstancesCount(ctx, virtualMachineScaleSetsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The current number of workers is: %d", currentWorkerCount))

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

		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaled worker VMSS to %d nodes", desiredWorkerCount))

		// Let's stay in the current state.
		return currentState, nil
	}

	// We didn't scale up the VMSS, ready to move to next step.
	return CordonOldWorkers, nil
}

func (r *Resource) scaleDownWorkerVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	// Scale down to the desired number of nodes in worker VMSS.
	desiredWorkerCount := *machinePool.Spec.Replicas
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaling worker VMSS to %d nodes", desiredWorkerCount))

	strategy := scalestrategy.Quick{}
	err = r.ScaleVMSS(ctx, virtualMachineScaleSetsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool), int64(desiredWorkerCount), strategy)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaled worker VMSS to %d nodes", desiredWorkerCount))

	// Enable cluster autoscaler for this nodepool.
	err = r.enableClusterAutoscaler(ctx, virtualMachineScaleSetsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	return DeploymentUninitialized, nil
}
