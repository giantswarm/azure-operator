package nodepool

import (
	"context"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/annotation"
)

const (
	// Types
	Stage = "Stage"

	// States
	CordonOldWorkers            = "CordonOldWorkers"
	DeploymentUninitialized     = ""
	DrainOldWorkerNodes         = "DrainOldWorkerNodes"
	ScaleUpWorkerVMSS           = "ScaleUpWorkerVMSS"
	ScaleDownWorkerVMSS         = "ScaleDownWorkerVMSS"
	TerminateOldWorkerInstances = "TerminateOldWorkerInstances"
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

	annotations := azureMachinePool.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[annotation.StateMachineCurrentState] = state

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

	annotations := azureMachinePool.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	status, exists := annotations[annotation.StateMachineCurrentState]
	if !exists {
		return "", nil
	}

	return status, nil
}
