package nodepool

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/reconciliationcanceledcontext"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/nodepool/template"
)

func (r *Resource) deploymentUninitializedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if machinePool == nil {
		return currentState, microerror.Mask(ownerReferenceNotSet)
	}

	if !machinePool.GetDeletionTimestamp().IsZero() {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "MachinePool is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "Cluster is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	azureCluster, err := r.getAzureClusterFromCluster(ctx, cluster)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if !azureCluster.GetDeletionTimestamp().IsZero() {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "AzureCluster is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	release, err := r.getReleaseFromMetadata(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	storageAccountsClient, err := r.ClientFactory.GetStorageAccountsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	// Compute desired state for Azure ARM Deployment.
	desiredDeployment, err := r.getDesiredDeployment(ctx, storageAccountsClient, release, machinePool, &azureMachinePool, cluster, azureCluster)
	if IsNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "Azure resource not found", "stack", microerror.JSON(err))
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	} else if IsSubnetNotReadyError(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "subnet is not Ready, it's probably still being created", "stack", microerror.JSON(err))
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	// Fetch current Azure ARM Deployment.
	currentDeployment, err := deploymentsClient.Get(ctx, key.ClusterID(&azureMachinePool), key.NodePoolDeploymentName(&azureMachinePool))
	if IsDeploymentNotFound(err) {
		// We haven't created the deployment just yet, it's fine.
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	defer func() {
		var currentProvisioningState *string
		if currentDeployment.Properties != nil && currentDeployment.Properties.ProvisioningState != nil {
			currentProvisioningState = currentDeployment.Properties.ProvisioningState
		}
		// Update DeploymentSucceeded Condition for this AzureMachinePool
		_ = r.UpdateDeploymentSucceededCondition(ctx, &azureMachinePool, currentProvisioningState)
	}()

	// Figure out if we need to submit the ARM Deployment.
	deploymentNeedsToBeSubmitted := currentDeployment.IsHTTPStatus(404)
	nodesNeedToBeRolled := false
	if !deploymentNeedsToBeSubmitted {
		changes, err := template.Diff(currentDeployment, desiredDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		// When customer is only scaling the cluster,
		// we don't need to move to the next state of the state machine which will rollout all the nodes.
		numberOfChangedParameters := len(changes)
		deploymentNeedsToBeSubmitted = numberOfChangedParameters > 0
		nodesNeedToBeRolled = numberOfChangedParameters > 1 || numberOfChangedParameters == 1 && !contains(changes, "scaling")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "Checking if deployment is out of date and needs to be re-submitted", "deploymentNeedsToBeSubmitted", deploymentNeedsToBeSubmitted, "nodesNeedToBeRolled", nodesNeedToBeRolled, "changedParameters", changes)
	}

	if deploymentNeedsToBeSubmitted {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")

		_, err = r.ensureDeployment(ctx, deploymentsClient, desiredDeployment, &azureMachinePool)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if nodesNeedToBeRolled {
			return ScaleUpWorkerVMSS, nil
		}

		return currentState, nil
	}

	// Potential states are: Succeeded, Failed, Canceled. All other values indicate the operation is still running.
	// https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/async-operations#provisioningstate-values
	switch *currentDeployment.Properties.ProvisioningState {
	case "Failed", "Canceled":
		r.Logger.LogCtx(ctx, "level", "debug", "message", "ARM deployment has failed, re-applying")
		r.Debugger.LogFailedDeployment(ctx, currentDeployment, err)

		err := r.saveAzureIDsInCR(ctx, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, &azureMachinePool)
		if err != nil {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "error trying to save object in k8s API", "stack", microerror.JSON(microerror.Mask(err)))
			r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return currentState, nil
		}

		// Deployment is not running and not succeeded (Failed?)
		// This indicates some kind of error in the deployment template and/or parameters.
		// Restart state machine on the next loop to apply the deployment once again.
		// (If the azure operator has been fixed/updated in the meantime that could lead to a fix).
		_, err = r.ensureDeployment(ctx, deploymentsClient, desiredDeployment, &azureMachinePool)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return currentState, nil
	default:
		r.Logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")

		err := r.saveAzureIDsInCR(ctx, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, &azureMachinePool)
		if err != nil {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "error trying to save object in k8s API", "stack", microerror.JSON(microerror.Mask(err)))
			r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return currentState, nil
		}

		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	}
}

func (r *Resource) saveAzureIDsInCR(ctx context.Context, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, azureMachinePool *capzexpv1alpha3.AzureMachinePool) error {
	r.Logger.LogCtx(ctx, "level", "debug", "message", "saving provider status info in CR")

	vmss, err := virtualMachineScaleSetsClient.Get(ctx, key.ClusterID(azureMachinePool), key.NodePoolVMSSName(azureMachinePool))
	if err != nil {
		return microerror.Mask(err)
	}

	instances, err := r.GetVMSSInstances(ctx, virtualMachineScaleSetVMsClient, key.ClusterID(azureMachinePool), key.NodePoolVMSSName(azureMachinePool))
	if err != nil {
		return microerror.Mask(err)
	}

	providerIDList := make([]string, len(instances))
	for i, vm := range instances {
		providerIDList[i] = fmt.Sprintf("azure://%s", *vm.ID)
	}
	azureMachinePool.Spec.ProviderIDList = providerIDList
	azureMachinePool.Spec.ProviderID = fmt.Sprintf("azure://%s", *vmss.ID)

	err = r.CtrlClient.Update(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.CtrlClient.Get(ctx, ctrlclient.ObjectKey{Name: azureMachinePool.Name, Namespace: azureMachinePool.Namespace}, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	provisioningState := capzv1alpha3.VMState(*vmss.ProvisioningState)
	azureMachinePool.Status.ProvisioningState = &provisioningState
	azureMachinePool.Status.Ready = provisioningState == "Succeeded"
	azureMachinePool.Status.Replicas = int32(len(instances))

	err = r.CtrlClient.Status().Update(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "saved provider status info in CR")

	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (r *Resource) ensureDeployment(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, desiredDeployment azureresource.Deployment, azureMachinePool *capzexpv1alpha3.AzureMachinePool) (azureresource.Deployment, error) {
	r.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	err := r.CreateARMDeployment(ctx, deploymentsClient, desiredDeployment, key.ClusterID(azureMachinePool), key.NodePoolDeploymentName(azureMachinePool))
	if err != nil {
		return desiredDeployment, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment")

	return desiredDeployment, nil
}
