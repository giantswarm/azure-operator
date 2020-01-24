package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
)

// configureStateMachine configures and returns state machine that is driven by
// EnsureCreated.
func (r *Resource) configureStateMachine() {
	sm := state.Machine{
		DeploymentUninitialized:  r.deploymentUninitializedTransition,
		DeploymentInitialized:    r.deploymentInitializedTransition,
		ProvisioningSuccessful:   r.provisioningSuccessfulTransition,
		MasterInstancesUpgrading: r.masterInstancesUpgradingTransition,
		WorkerInstancesUpgrading: r.workerInstancesUpgradingTransition,
		DeploymentCompleted:      r.deploymentCompletedTransition,
	}

	r.stateMachine = sm
}

// EnsureCreated operates in 3 different stages which are executed sequentially.
// The first stage is for uploading ARM templates and is represented by stage
// DeploymentInitialized.
// The second stage is for waiting for ARM templates to be applied and is represented
// by stage ProvisioningSuccessful.
// The third stage is for draining and upgrading the VMSS instances and is represented by
// stage InstancesUpgrading.
// Once all instances are Upgraded the state becomes DeploymentCompleted and the reconciliation
// loop stops until a change in the ARM template or parameters is detected.
// Check docs/instances-stages-v13.svg file for a grafical representation of this process.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var newState state.State
	var currentState state.State
	{
		s, err := r.getResourceStatus(customObject, Stage)
		if err != nil {
			return microerror.Mask(err)
		}
		currentState = state.State(s)

		if currentState == "" {
			// DeploymentUninitialized is the initial state for instance resource.
			newState = DeploymentUninitialized
			r.logger.LogCtx(ctx, "level", "debug", "message", "no current state present")
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("current state: %s", currentState))
			newState, err = r.stateMachine.Execute(ctx, obj, currentState)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	if newState != currentState {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("new state: %s", newState))
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting resource status to '%s/%s'", Stage, newState))
		err = r.setResourceStatus(customObject, Stage, string(newState))
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set resource status to '%s/%s'", Stage, newState))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "no state change")
	}

	return nil
}
