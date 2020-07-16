package nodepool

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes/scalestrategy"
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

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, *cluster)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	allReady, err := vmsscheck.InstancesAreRunning(ctx, r.Logger, virtualMachineScaleSetVMsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if vmsscheck.IsVMSSUnsafeError(err) {
		// VMSS rate limits are not safe, let's wait for next reconciliation loop.
		return currentState, nil
	} else if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	// Not all workers are Running in Azure, wait for next reconciliation loop.
	if !allReady {
		return currentState, nil
	}

	strategy := scalestrategy.Quick{}

	// All workers ready, we can scale up if needed.
	desiredWorkerCount := int64(*machinePool.Spec.Replicas * 2)
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The desired number of workers is: %d", desiredWorkerCount))

	currentWorkerCount, err := r.GetInstancesCount(ctx, virtualMachineScaleSetsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The current number of workers is: %d", currentWorkerCount))

	if desiredWorkerCount > currentWorkerCount {
		err = r.ScaleVMSS(ctx, virtualMachineScaleSetsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool), desiredWorkerCount, strategy)
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}

		r.InstanceWatchdog.GuardVMSS(ctx, virtualMachineScaleSetVMsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
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

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, *cluster)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(credentialSecret.Namespace, credentialSecret.Name)
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

	return DeploymentUninitialized, nil
}
