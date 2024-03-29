package nodepool

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v8/service/controller/azuremachinepool/handler/nodepool/template"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
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
		r.Logger.Debugf(ctx, "MachinePool is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.Logger.Debugf(ctx, "Cluster is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	azureCluster, err := r.getAzureClusterFromCluster(ctx, cluster)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if !azureCluster.GetDeletionTimestamp().IsZero() {
		r.Logger.Debugf(ctx, "AzureCluster is being deleted, skipping reconciling node pool")
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

	vmss, err := virtualMachineScaleSetsClient.Get(ctx, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if IsNotFound(err) {
		// We haven't created the VMSS just yet, it's fine.
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	// Compute desired state for Azure ARM Deployment.
	desiredDeployment, err := r.getDesiredDeployment(ctx, storageAccountsClient, release, machinePool, &azureMachinePool, azureCluster, vmss)
	if IsNotFound(err) {
		r.Logger.Debugf(ctx, "Azure resource not found")
		r.Logger.Debugf(ctx, "canceling resource")
		return currentState, nil
	} else if IsSubnetNotReadyError(err) {
		r.Logger.Debugf(ctx, "subnet is not Ready, it's probably still being created")
		r.Logger.Debugf(ctx, microerror.JSON(err))
		r.Logger.Debugf(ctx, "canceling resource")
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

	// Figure out if we need to submit the ARM Deployment.
	deploymentNeedsToBeSubmitted := currentDeployment.IsHTTPStatus(http.StatusNotFound)
	nodesNeedToBeRolled := false
	if !deploymentNeedsToBeSubmitted {
		changes, err := template.Diff(currentDeployment, desiredDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if !contains(changes, "scaling") {
			desiredParameters, err := template.NewFromDeployment(desiredDeployment)
			if err != nil {
				return currentState, microerror.Mask(err)
			}

			if desiredParameters.Scaling.MinReplicas == desiredParameters.Scaling.MaxReplicas &&
				vmss.Sku != nil &&
				vmss.Sku.Capacity != nil &&
				desiredParameters.Scaling.CurrentReplicas != int32(*vmss.Sku.Capacity) {
				changes = append(changes, "scaling")
			}
		}

		// When customer is only scaling the cluster,
		// we don't need to move to the next state of the state machine which will rollout all the nodes.
		numberOfChangedParameters := len(changes)
		deploymentNeedsToBeSubmitted = numberOfChangedParameters > 0
		nodesNeedToBeRolled = numberOfChangedParameters > 1 || numberOfChangedParameters == 1 && !contains(changes, "scaling")
		r.Logger.Debugf(ctx, "Checking if deployment is out of date and needs to be re-submitted", "deploymentNeedsToBeSubmitted", deploymentNeedsToBeSubmitted, "nodesNeedToBeRolled", nodesNeedToBeRolled, "changedParameters", changes)
	}

	if deploymentNeedsToBeSubmitted {
		r.Logger.Debugf(ctx, "template or parameters changed")

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
		r.Logger.Debugf(ctx, "ARM deployment has failed, re-applying")
		r.Debugger.LogFailedDeployment(ctx, currentDeployment, err)

		err := r.saveAzureIDsInCR(ctx, virtualMachineScaleSetVMsClient, &azureMachinePool, vmss)
		if apierrors.IsConflict(microerror.Cause(err)) {
			r.Logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
			r.Logger.Debugf(ctx, "canceling resource")
			return currentState, nil
		} else if err != nil {
			return currentState, microerror.Mask(err)
		}

		// Deployment is not running and not succeeded (Failed?)
		// This indicates some kind of error in the deployment template and/or parameters.
		// Restart state machine on the next loop to apply the deployment once again.
		// (If the azure operator has been fixed/updated in the meantime that could lead to a fix).
		_, err = r.ensureDeployment(ctx, deploymentsClient, desiredDeployment, &azureMachinePool)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		r.Logger.Debugf(ctx, "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return currentState, nil
	default:
		r.Logger.Debugf(ctx, "template and parameters unchanged")

		err := r.saveAzureIDsInCR(ctx, virtualMachineScaleSetVMsClient, &azureMachinePool, vmss)
		if apierrors.IsConflict(microerror.Cause(err)) {
			r.Logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
			r.Logger.Debugf(ctx, "canceling resource")
			return currentState, nil
		} else if err != nil {
			return currentState, microerror.Mask(err)
		}

		r.Logger.Debugf(ctx, "canceling resource")
		return currentState, nil
	}
}

func (r *Resource) saveAzureIDsInCR(ctx context.Context, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, azureMachinePool *capzexp.AzureMachinePool, vmss compute.VirtualMachineScaleSet) error {
	if vmss.IsHTTPStatus(http.StatusNotFound) {
		return nil
	}

	r.Logger.Debugf(ctx, "saving provider status info in CR")

	instances, err := r.GetVMSSInstances(ctx, *azureMachinePool)
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

	provisioningState := capz.ProvisioningState(*vmss.ProvisioningState)
	azureMachinePool.Status.ProvisioningState = &provisioningState
	azureMachinePool.Status.Ready = provisioningState == "Succeeded"
	azureMachinePool.Status.Replicas = int32(len(instances))

	err = r.CtrlClient.Status().Update(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.Debugf(ctx, "saved provider status info in CR")

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

func (r *Resource) ensureDeployment(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, desiredDeployment azureresource.Deployment, azureMachinePool *capzexp.AzureMachinePool) (azureresource.Deployment, error) {
	r.Logger.Debugf(ctx, "ensuring deployment")

	err := r.CreateARMDeployment(ctx, deploymentsClient, desiredDeployment, key.ClusterID(azureMachinePool), key.NodePoolDeploymentName(azureMachinePool))
	if err != nil {
		return desiredDeployment, microerror.Mask(err)
	}

	r.Logger.Debugf(ctx, "ensured deployment")

	return desiredDeployment, nil
}
