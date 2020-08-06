package instance

import (
	"context"
	"fmt"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

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
			DeploymentUninitialized:        r.deploymentUninitializedTransition,
			DeploymentInitialized:          r.deploymentInitializedTransition,
			ProvisioningSuccessful:         r.provisioningSuccessfulTransition,
			ClusterUpgradeRequirementCheck: r.clusterUpgradeRequirementCheckTransition,
			ScaleUpWorkerVMSS:              r.scaleUpWorkerVMSSTransition,
			CordonOldWorkers:               r.cordonOldWorkersTransition,
			WaitForWorkersToBecomeReady:    r.waitForWorkersToBecomeReadyTransition,
			DrainOldWorkerNodes:            r.drainOldWorkerNodesTransition,
			TerminateOldWorkerInstances:    r.terminateOldWorkersTransition,
			ScaleDownWorkerVMSS:            r.scaleDownWorkerVMSSTransition,
			DeploymentCompleted:            r.deploymentCompletedTransition,
		},
	}

	return sm
}

// This resource applies the ARM template for the worker instances, monitors the process and handles upgrades.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if key.WorkerCount(cr) == 0 {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "no built-in workers defined")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	if isMasterUpgrading(cr) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "master is upgrading")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
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
		reconciliationcanceledcontext.SetCanceled(ctx)
	} else {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "no state change")
	}

	return nil
}

func isMasterUpgrading(cr providerv1alpha1.AzureConfig) bool {
	var status string
	{
		for _, r := range cr.Status.Cluster.Resources {
			if r.Name != masters.Name {
				continue
			}

			for _, c := range r.Conditions {
				if c.Type == Stage {
					status = c.Status
				}
			}
		}
	}

	return status != DeploymentCompleted
}
