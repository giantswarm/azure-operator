package masters

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v7/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v7/service/controller/key"
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

		r.Logger.Debugf(ctx, "current state: %s", currentState)
		newState, err = r.StateMachine.Execute(ctx, obj, currentState)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if newState != currentState {
		r.Logger.Debugf(ctx, "new state: %s", newState)
		r.Logger.Debugf(ctx, "setting resource status to '%s/%s'", Stage, newState)
		err = r.SetResourceStatus(ctx, cr, Stage, string(newState))
		if err != nil {
			return microerror.Mask(err)
		}
		r.Logger.Debugf(ctx, "set resource status to '%s/%s'", Stage, newState)
		r.Logger.Debugf(ctx, "canceling reconciliation")
	} else {
		r.Logger.Debugf(ctx, "no state change")
	}

	return nil
}
