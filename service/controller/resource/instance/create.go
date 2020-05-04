package instance

import (
	"context"
	"fmt"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/resource/masters"
)

// configureStateMachine configures and returns state machine that is driven by
// EnsureCreated.
func (r *Resource) configureStateMachine() {
	sm := state.Machine{
		DeploymentUninitialized:        r.deploymentUninitializedTransition,
		DeploymentInitialized:          r.deploymentInitializedTransition,
		ProvisioningSuccessful:         r.provisioningSuccessfulTransition,
		ClusterUpgradeRequirementCheck: r.clusterUpgradeRequirementCheckTransition,
		ScaleUpWorkerVMSS:              r.scaleUpWorkerVMSSTransition,

		WaitNewVMSSWorkers: r.waitNewVMSSWorkersTransition,

		CordonOldVMSS:    r.cordonOldVMSSTransition,
		CordonOldWorkers: r.cordonOldWorkersTransition,

		WaitForWorkersToBecomeReady: r.waitForWorkersToBecomeReadyTransition,

		DrainOldVMSS:        r.drainOldVMSSTransition,
		DrainOldWorkerNodes: r.drainOldWorkerNodesTransition,

		TerminateOldVMSS:            r.terminateOldVmssTransition,
		TerminateOldWorkerInstances: r.terminateOldWorkersTransition,

		ScaleDownWorkerVMSS: r.scaleDownWorkerVMSSTransition,
		DeploymentCompleted: r.deploymentCompletedTransition,
	}

	r.stateMachine = sm
}

// This resource applies the ARM template for the worker instances, monitors the process and handles upgrades.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if isMasterUpgrading(cr) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "master is upgrading")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		return nil
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
		reconciliationcanceledcontext.SetCanceled(ctx)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "no state change")
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
