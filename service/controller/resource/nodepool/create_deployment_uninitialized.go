package nodepool

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/reconciliationcanceledcontext"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodepool/template"
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

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	desiredDeployment, err := r.getDesiredDeployment(ctx, storageAccountsClient, release, cluster, azureCluster, machinePool, &azureMachinePool)
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

	deploymentIsOutOfDate, err := template.IsOutOfDate(currentDeployment, desiredDeployment)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	nodesNeedToBeRolled, err := template.NeedToRolloutNodes(currentDeployment, desiredDeployment)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "message", "Checking if deployment is out of date", "outOfDate", deploymentIsOutOfDate, "nodesWillBeRolledOut", nodesNeedToBeRolled)

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
	provisioningState := capzv1alpha3.VMState(*currentDeployment.Properties.ProvisioningState)
	azureMachinePool.Status.ProvisioningState = &provisioningState
	switch *currentDeployment.Properties.ProvisioningState {
	case "Failed", "Canceled":
		r.Debugger.LogFailedDeployment(ctx, currentDeployment, err)

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
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	}
}

func (r *Resource) getDesiredDeployment(ctx context.Context, storageAccountsClient *storage.AccountsClient, release *releasev1alpha1.Release, cluster *capiv1alpha3.Cluster, azureCluster *capzv1alpha3.AzureCluster, machinePool *capiexpv1alpha3.MachinePool, azureMachinePool *capzexpv1alpha3.AzureMachinePool) (azureresource.Deployment, error) {
	desiredDeployment, err := r.newDeployment(ctx, storageAccountsClient, release, machinePool, azureMachinePool, cluster, azureCluster)
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
