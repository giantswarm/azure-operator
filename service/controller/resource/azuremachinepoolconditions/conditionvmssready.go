package azuremachinepoolconditions

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	azureconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	VMSSNotFoundReason                 = "VMSSNotFound"
	VMSSProvisioningStatePrefix        = "VMSSProvisioningState"
	VMSSProvisioningStateUnknownReason = "VMSSProvisioningStateUnknown"
	VMSSProvisioningStateSucceeded     = string(compute.ProvisioningStateSucceeded)
	VMSSProvisioningStateFailed        = string(compute.ProvisioningStateFailed)
)

func (r *Resource) ensureVMSSReadyCondition(ctx context.Context, azureMachinePool *capzexp.AzureMachinePool) error {
	r.logger.Debugf(ctx, "ensuring condition %s", azureconditions.VMSSReadyCondition)

	deploymentsClient, err := r.azureClientsFactory.GetDeploymentsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	// Now let's first check ARM deployment state
	deploymentName := key.NodePoolDeploymentName(azureMachinePool)
	isDeploymentSuccessful, err := r.deploymentChecker.CheckIfDeploymentIsSuccessful(ctx, deploymentsClient, azureMachinePool, deploymentName, azureconditions.VMSSReadyCondition)
	if err != nil {
		return microerror.Mask(err)
	} else if !isDeploymentSuccessful {
		// Function checkIfDeploymentIsSuccessful that is called above, if it
		// sees that the deployment is not succeeded, for whatever reason, it
		// will also set appropriate condition value, so our job here is done.
		return nil
	}

	// Deployment is successful, we proceed with checking the actual Azure
	// VMSS.
	vmssClient, err := r.azureClientsFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroupName := key.ClusterName(azureMachinePool)
	vmssName := key.NodePoolVMSSName(azureMachinePool)

	vmss, err := vmssClient.Get(ctx, resourceGroupName, vmssName)
	if IsNotFound(err) {
		r.setVMSSNotFound(ctx, azureMachinePool, vmssName, azureconditions.VMSSReadyCondition)
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	// Note: Here we are only checking the provisioning state of VMSS. Ideally
	// we would check the provisioning and power state of all instances, but
	// that would require more VMSS instance API calls that have very low
	// throttling limits, so we will add that later, once throttling situation
	// is better.

	// Check if VMSS provisioning state is set. We expect that it is, since we
	// already checked the deployment, but it's not impossible that the VMSS
	// resource got changed for some reason.
	if vmss.ProvisioningState == nil {
		r.setVMSSProvisioningStateUnknown(ctx, azureMachinePool, deploymentName, azureconditions.VMSSReadyCondition)
		return nil
	}

	switch *vmss.ProvisioningState {
	// VMSS provisioning state is Succeeded, all good.
	case VMSSProvisioningStateSucceeded:
		capiconditions.MarkTrue(azureMachinePool, azureconditions.VMSSReadyCondition)
	// VMSS provisioning state is Failed, VMSS has some issues.
	case VMSSProvisioningStateFailed:
		r.setVMSSProvisioningStateFailed(ctx, azureMachinePool, vmssName, azureconditions.VMSSReadyCondition)
	default:
		// VMSS provisioning state not Succeeded, set current state to VMSSReady condition.
		r.setVMSSProvisioningStateWarning(ctx, azureMachinePool, vmssName, *vmss.ProvisioningState, azureconditions.VMSSReadyCondition)
	}

	// Log current VMSSReady condition
	r.logConditionStatus(ctx, azureMachinePool, azureconditions.VMSSReadyCondition)
	r.logger.Debugf(ctx, "ensured condition %s", azureconditions.VMSSReadyCondition)
	return nil
}

func (r *Resource) setVMSSNotFound(ctx context.Context, cr capiconditions.Setter, vmssName string, condition capi.ConditionType) {
	message := "VMSS %s is not found, which should not happen when the deployment is successful"
	messageArgs := vmssName
	capiconditions.MarkFalse(
		cr,
		condition,
		VMSSNotFoundReason,
		capi.ConditionSeverityError,
		message,
		messageArgs)

	r.logger.Debugf(ctx, message, messageArgs)
}

func (r *Resource) setVMSSProvisioningStateUnknown(ctx context.Context, cr capiconditions.Setter, deploymentName string, condition capi.ConditionType) {
	message := "VMSS %s provisioning state not returned by Azure API, check back in few minutes"
	messageArgs := deploymentName
	capiconditions.MarkFalse(
		cr,
		condition,
		VMSSProvisioningStateUnknownReason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logger.Debugf(ctx, message, messageArgs)
}

func (r *Resource) setVMSSProvisioningStateFailed(ctx context.Context, cr capiconditions.Setter, vmssName string, condition capi.ConditionType) {
	message := "VMSS %s failed, it might succeed after retrying, see Azure portal for more details"
	messageArgs := vmssName
	reason := VMSSProvisioningStatePrefix + VMSSProvisioningStateFailed

	capiconditions.MarkFalse(
		cr,
		condition,
		reason,
		capi.ConditionSeverityError,
		message,
		messageArgs)

	r.logger.Debugf(ctx, message, messageArgs)
}

func (r *Resource) setVMSSProvisioningStateWarning(ctx context.Context, cr capiconditions.Setter, vmssName string, currentProvisioningState string, condition capi.ConditionType) {
	message := "Deployment %s has not succeeded yet, current state is %s, " +
		"check back in few minutes, see Azure portal for more details"
	messageArgs := []interface{}{vmssName, currentProvisioningState}
	reason := VMSSProvisioningStatePrefix + currentProvisioningState

	capiconditions.MarkFalse(
		cr,
		condition,
		reason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs...)

	r.logger.Debugf(ctx, message, messageArgs...)
}
