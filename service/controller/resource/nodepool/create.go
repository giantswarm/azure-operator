package nodepool

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/azure-operator/v4/pkg/annotation"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/masters"
)

// createStateMachine configures and returns state machine that is driven by
// EnsureCreated.
func (r *Resource) createStateMachine() state.Machine {
	sm := state.Machine{
		Logger:       r.Logger,
		ResourceName: Name,
		Transitions: state.TransitionMap{
			DeploymentUninitialized:     r.deploymentUninitializedTransition,
			ScaleUpWorkerVMSS:           r.scaleUpWorkerVMSSTransition,
			CordonOldWorkers:            r.cordonOldWorkersTransition,
			WaitForWorkersToBecomeReady: r.waitForWorkersToBecomeReadyTransition,
			DrainOldWorkerNodes:         r.drainOldWorkerNodesTransition,
			TerminateOldWorkerInstances: r.terminateOldWorkersTransition,
			ScaleDownWorkerVMSS:         r.scaleDownWorkerVMSSTransition,
		},
	}

	return sm
}

// This resource applies the ARM template for the worker instances, monitors the process and handles upgrades.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if isMasterUpgrading(&azureMachinePool) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "master is upgrading")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	var newState state.State
	var currentState state.State
	{
		s, err := r.getCurrentState(ctx, azureMachinePool)
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

	err = r.CtrlClient.Status().Update(ctx, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	if newState != currentState {
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("new state: %s", newState))
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting resource status to %#q", newState))
		err = r.saveCurrentState(ctx, azureMachinePool, string(newState))
		if err != nil {
			return microerror.Mask(err)
		}
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set resource status to %#q", newState))
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
	} else {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "no state change")
	}

	return nil
}

func isMasterUpgrading(getter key.AnnotationsGetter) bool {
	masterUpgrading, exists := getter.GetAnnotations()[annotation.IsMasterUpgrading]
	if !exists {
		return false
	}

	return masterUpgrading != masters.DeploymentCompleted
}
