package nodepool

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/reconciliationcanceledcontext"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
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

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, *cluster)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	azureCluster, err := r.getAzureClusterFromCluster(ctx, cluster)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	release, err := r.getReleaseFromMetadata(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	storageAccountsClient, err := r.ClientFactory.GetStorageAccountsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	desiredDeployment, err := r.getDesiredDeployment(ctx, storageAccountsClient, release, azureCluster, machinePool, &azureMachinePool)
	if IsNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "Azure resource not found, canceling resource")
		return currentState, nil
	} else if IsSubnetNotReadyError(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "subnet is not Ready, it's probably still being created")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	} else if err != nil {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, microerror.Mask(err)
	}

	currentDeployment, err := deploymentsClient.Get(ctx, key.ClusterID(&azureMachinePool), key.NodePoolDeploymentName(&azureMachinePool))
	if IsDeploymentNotFound(err) {
		// We haven't created the deployment just yet, it's fine.
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	deploymentIsOutOfDate, nodesNeedToBeRolled, err := r.deploymentIsOutOfDate(ctx, currentDeployment, desiredDeployment)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if deploymentIsOutOfDate {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")

		_, err = r.ensureDeployment(ctx, deploymentsClient, desiredDeployment, &azureMachinePool)
		if err != nil {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return currentState, microerror.Mask(err)
		}

		if nodesNeedToBeRolled {
			return ScaleUpWorkerVMSS, nil
		}

		return currentState, nil
	}

	// Start watcher on the instances to avoid stuck VMs to block the deployment progress forever
	r.InstanceWatchdog.DeleteFailedVMSS(ctx, virtualMachineScaleSetVMsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))

	// Potential states are: Succeeded, Failed, Canceled. All other values indicate the operation is still running.
	// https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/async-operations#provisioningstate-values
	switch *currentDeployment.Properties.ProvisioningState {
	case "Failed", "Canceled":
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
			r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
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

	provisioningState := capzv1alpha3.VMState(*vmss.ProvisioningState)
	azureMachinePool.Status.ProvisioningState = &provisioningState
	azureMachinePool.Status.Replicas = int32(len(instances))
	azureMachinePool.Spec.ProviderID = fmt.Sprintf("azure://%s", *vmss.ID)

	providerIDList := make([]string, len(instances))
	for i, vm := range instances {
		providerIDList[i] = fmt.Sprintf("azure://%s", *vm.ID)
	}
	azureMachinePool.Spec.ProviderIDList = providerIDList

	err = r.CtrlClient.Update(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.CtrlClient.Status().Update(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "saved provider status info in CR")

	return nil
}

// deploymentIsOutOfDate decides whether or not we need to re-apply the ARM deployment template.
// There are two cases where we want to update the cluster:
// - customer has decided to update to a newer GiantSwarm release
// - customer has changed some configuration and we need to apply it
func (r *Resource) deploymentIsOutOfDate(ctx context.Context, currentDeployment azureresource.DeploymentExtended, desiredDeployment azureresource.Deployment) (bool, bool, error) {
	if currentDeployment.IsHTTPStatus(404) {
		return true, false, nil
	}

	currentDeploymentParameters, ok := currentDeployment.Properties.Parameters.(map[string]interface{})
	if !ok {
		return false, false, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, currentDeployment.Properties.Parameters)
	}

	desiredDeploymentParameters, ok := desiredDeployment.Properties.Parameters.(map[string]interface{})
	if !ok {
		return false, false, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, currentDeployment.Properties.Parameters)
	}

	currentAzureMachinePoolVersion := castCurrent(currentDeploymentParameters["azureMachinePoolVersion"])
	desiredAzureMachinePoolVersion := castDesired(desiredDeploymentParameters["azureMachinePoolVersion"])
	customerHasChangedConfiguration := currentAzureMachinePoolVersion != desiredAzureMachinePoolVersion

	currentMachinePoolVersion := castCurrent(currentDeploymentParameters["machinePoolVersion"])
	desiredMachinePoolVersion := castDesired(desiredDeploymentParameters["machinePoolVersion"])
	customerHasScaledTheCluster := currentMachinePoolVersion != desiredMachinePoolVersion

	currentAzureOperatorVersion := castCurrent(currentDeploymentParameters["azureOperatorVersion"])
	desiredAzureOperatorVersion := castDesired(desiredDeploymentParameters["azureOperatorVersion"])
	customerIsUpgradingTheCluster := currentAzureOperatorVersion != desiredAzureOperatorVersion

	r.Logger.LogCtx(ctx, "message", "Checking if deployment is out of date",
		"customerHasChangedConfiguration", customerHasChangedConfiguration,
		"customerIsUpgradingTheCluster", customerIsUpgradingTheCluster,
		"customerHasScaledTheCluster", customerHasScaledTheCluster,
		"nodesWillBeRolledOut", customerIsUpgradingTheCluster || customerHasChangedConfiguration,
		"currentAzureOperatorVersion", currentAzureOperatorVersion,
		"desiredAzureOperatorVersion", desiredAzureOperatorVersion,
		"currentAzureMachinePoolVersion", currentAzureMachinePoolVersion,
		"desiredAzureMachinePoolVersion", desiredAzureMachinePoolVersion,
		"currentMachinePoolVersion", currentMachinePoolVersion,
		"desiredMachinePoolVersion", desiredMachinePoolVersion)

	return customerHasChangedConfiguration || customerIsUpgradingTheCluster || customerHasScaledTheCluster, customerIsUpgradingTheCluster || customerHasChangedConfiguration, nil
}

func castCurrent(param interface{}) string {
	return param.(map[string]interface{})["value"].(string)
}

func castDesired(param interface{}) string {
	return param.(struct{ Value interface{} }).Value.(string)
}

func (r *Resource) getDesiredDeployment(ctx context.Context, storageAccountsClient *storage.AccountsClient, release *releasev1alpha1.Release, azureCluster *capzv1alpha3.AzureCluster, machinePool *capiexpv1alpha3.MachinePool, azureMachinePool *capzexpv1alpha3.AzureMachinePool) (azureresource.Deployment, error) {
	desiredDeployment, err := r.newDeployment(ctx, storageAccountsClient, release, machinePool, azureMachinePool, azureCluster)
	if controllercontext.IsInvalidContext(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", err.Error())
		r.Logger.LogCtx(ctx, "level", "debug", "message", "missing dispatched output values in controller context")
		return azureresource.Deployment{}, microerror.Mask(err)
	} else if blobclient.IsBlobNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
		return azureresource.Deployment{}, microerror.Mask(err)
	} else if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	return desiredDeployment, nil
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
