package nodepool

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/reconciliationcanceledcontext"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
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
			WaitForWorkersToBecomeReady: r.waitForWorkersToBecomeReadyTransition,
			CordonOldWorkerInstances:    r.cordonOldWorkerInstances,
			DrainOldWorkerInstances:     r.drainOldWorkerInstances,
			TerminateOldWorkerInstances: r.terminateOldWorkersTransition,
		},
	}

	return sm
}

// EnsureCreated will create an ARM deployment for every node pool.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	upgrading, err := r.isMasterUpgrading(ctx, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}
	if upgrading {
		r.Logger.Debugf(ctx, "master is upgrading")
		r.Logger.Debugf(ctx, "canceling resource")
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

		r.Logger.Debugf(ctx, "current state: %s", currentState)
		newState, err = r.StateMachine.Execute(ctx, obj, currentState)
		if state.IsUnkownStateError(err) {
			// This can happen if there is a race condition with a previous version of the azure operator
			// or if the node pool at upgrade time was in a state that doesn't exists any more in this azure
			// operator version.
			// At this stage if this error happened while upgrading to a new release and the ARM deployment was already applied
			// we need to ensure nodes are going to be rolled out.
			// We move directly to `ScaleUpWorkerVMSS`. If for any reason the ARM deployment is not applied then the
			// `ScaleUpWorkerVMSS` handler will detect the situation and go back to the `DeploymentUninitialized` state.
			r.Logger.Debugf(ctx, "Azure Machine Pool was in state %q that is unknown to this azure operator version's state machine. To avoid blocking an upgrade the state will be set to %q.", currentState, ScaleUpWorkerVMSS)
			newState = ScaleUpWorkerVMSS
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if newState != currentState {
		r.Logger.Debugf(ctx, "new state: %s", newState)
		r.Logger.Debugf(ctx, "setting resource status to %#q", newState)
		err = r.saveCurrentState(ctx, azureMachinePool, string(newState))
		if apierrors.IsConflict(microerror.Cause(err)) {
			r.Logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
			r.Logger.Debugf(ctx, "no state change")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
		r.Logger.Debugf(ctx, "set resource status to %#q", newState)
		r.Logger.Debugf(ctx, "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
	} else {
		r.Logger.Debugf(ctx, "no state change")
	}

	return nil
}

func (r *Resource) isMasterUpgrading(ctx context.Context, amp *v1alpha3.AzureMachinePool) (bool, error) {
	r.Logger.Debugf(ctx, "Checking if master nodes are upgrading")

	// Get a list of all nodes, including worker nodes.
	nodeList := v1.NodeList{}
	err := r.CtrlClient.List(ctx, &nodeList)
	if err != nil {
		return false, microerror.Mask(err)
	}

	for _, node := range nodeList.Items {
		if node.Labels["role"] == "master" || node.Labels["kubernetes.io/role"] == "master" {
			// Check if node has the right azure operator version label
			if node.Labels[label.AzureOperatorVersion] != project.Version() {
				r.Logger.Debugf(ctx, "Node %q is not running azure operator version %q (it has %q)", node.Name, project.Version(), node.Labels[label.AzureOperatorVersion])
				return false, nil
			}
			if !isReady(node) {
				r.Logger.Debugf(ctx, "Node %q is not ready", node.Name)
				return false, nil
			}
		}
	}

	return true, nil
}
