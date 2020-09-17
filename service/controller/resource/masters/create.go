package masters

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

// createStateMachine configures and returns state machine that is driven by
// EnsureCreated.
func (r *Resource) createStateMachine() state.Machine {
	sm := state.Machine{
		Logger:       r.Logger,
		ResourceName: Name,
		Transitions: state.TransitionMap{
			Empty:                          r.emptyStateTransition,
			DeploymentUninitialized:        r.deploymentUninitializedTransition,
			DeploymentInitialized:          r.deploymentInitializedTransition,
			ProvisioningSuccessful:         r.provisioningSuccessfulTransition,
			ClusterUpgradeRequirementCheck: r.clusterUpgradeRequirementCheckTransition,
			MasterInstancesUpgrading:       r.masterInstancesUpgradingTransition,
			WaitForMastersToBecomeReady:    r.waitForMastersToBecomeReadyTransition,
			DeploymentCompleted:            r.deploymentCompletedTransition,
		},
	}

	return sm
}

// This resource applies the ARM template for the master instances, monitors the process and handles upgrades.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var newState state.State
	var currentState state.State
	{
		s, err := r.GetResourceStatus(ctx, cr, Stage)
		if err != nil {
			return microerror.Mask(err)
		}
		currentState = state.State(s)

		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("current state: %s", currentState))
		newState, err = r.StateMachine.Execute(ctx, obj, currentState)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if newState != currentState {
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("new state: %s", newState))
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting resource status to '%s/%s'", Stage, newState))
		err = r.SetResourceStatus(ctx, cr, Stage, string(newState))
		if err != nil {
			return microerror.Mask(err)
		}
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set resource status to '%s/%s'", Stage, newState))
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
	} else {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "no state change")
	}

	return nil
}
