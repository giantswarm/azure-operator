package masters

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

// configureStateMachine configures and returns state machine that is driven by
// EnsureCreated.
func (r *Resource) configureStateMachine() {
	sm := state.Machine{
		DeploymentUninitialized: r.deploymentUninitializedTransition,
		//DeploymentInitialized:          r.deploymentInitializedTransition,
		//ProvisioningSuccessful:         r.provisioningSuccessfulTransition,
		//ClusterUpgradeRequirementCheck: r.clusterUpgradeRequirementCheckTransition,
		//MasterInstancesUpgrading:       r.masterInstancesUpgradingTransition,
		//WaitForMastersToBecomeReady:    r.waitForMastersToBecomeReadyTransition,
		//DeploymentCompleted:            r.deploymentCompletedTransition,
	}

	r.stateMachine = sm
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
		s, err := r.getResourceStatus(cr, Stage)
		if err != nil {
			return microerror.Mask(err)
		}
		currentState = state.State(s)

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("current state: %s", currentState))
		newState, err = r.stateMachine.Execute(ctx, obj, currentState)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if newState != currentState {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("new state: %s", newState))
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting resource status to '%s/%s'", Stage, newState))
		err = r.setResourceStatus(cr, Stage, string(newState))
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set resource status to '%s/%s'", Stage, newState))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "no state change")
	}

	// We should wait and avoid further resources to reconciliate as long as the masters are not successfully deployed.
	if newState != DeploymentCompleted {
		reconciliationcanceledcontext.SetCanceled(ctx)
	}

	return nil
}
