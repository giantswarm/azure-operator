package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v6/pkg/annotation"
)

const (
	// Types
	Stage = "Stage"

	// States
	DeploymentUninitialized     = ""
	ScaleUpWorkerVMSS           = "ScaleUpWorkerVMSS" // nolint:gosec
	CordonOldWorkerInstances    = "CordonOldWorkerInstances"
	DrainOldWorkerInstances     = "DrainOldWorkerInstances"
	TerminateOldWorkerInstances = "TerminateOldWorkerInstances"
	WaitForWorkersToBecomeReady = "WaitForWorkersToBecomeReady"
)

func (r *Resource) saveCurrentState(ctx context.Context, customObject capzexp.AzureMachinePool, state string) error {
	// Get the newest CR version. Otherwise status update may fail because of:
	//
	//	 the object has been modified; please apply your changes to the
	//	 latest version and try again
	//
	azureMachinePool := &capzexp.AzureMachinePool{}
	err := r.CtrlClient.Get(ctx, client.ObjectKey{Namespace: customObject.Namespace, Name: customObject.Name}, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	if azureMachinePool.Annotations == nil {
		azureMachinePool.Annotations = map[string]string{}
	}

	azureMachinePool.Annotations[annotation.StateMachineCurrentState] = state

	err = r.CtrlClient.Update(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) getCurrentState(ctx context.Context, customObject capzexp.AzureMachinePool) (string, error) {
	azureMachinePool := &capzexp.AzureMachinePool{}
	err := r.CtrlClient.Get(ctx, client.ObjectKey{Namespace: customObject.Namespace, Name: customObject.Name}, azureMachinePool)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if azureMachinePool.Annotations == nil {
		azureMachinePool.Annotations = map[string]string{}
	}

	status, exists := azureMachinePool.Annotations[annotation.StateMachineCurrentState]
	if !exists {
		return "", nil
	}

	return status, nil
}
