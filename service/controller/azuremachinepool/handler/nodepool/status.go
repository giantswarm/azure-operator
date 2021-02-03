package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/annotation"
)

const (
	// Types
	Stage = "Stage"

	// States
	DeploymentUninitialized     = ""
	ScaleUpWorkerVMSS           = "ScaleUpWorkerVMSS"
	TerminateOldWorkerInstances = "TerminateOldWorkerInstances"
	WaitForOldWorkersToBeGone   = "WaitForOldWorkersToBeGone"
	WaitForWorkersToBecomeReady = "WaitForWorkersToBecomeReady"
)

func (r *Resource) saveCurrentState(ctx context.Context, customObject v1alpha3.AzureMachinePool, state string) error {
	// Get the newest CR version. Otherwise status update may fail because of:
	//
	//	 the object has been modified; please apply your changes to the
	//	 latest version and try again
	//
	azureMachinePool := &v1alpha3.AzureMachinePool{}
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

func (r *Resource) getCurrentState(ctx context.Context, customObject v1alpha3.AzureMachinePool) (string, error) {
	azureMachinePool := &v1alpha3.AzureMachinePool{}
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
