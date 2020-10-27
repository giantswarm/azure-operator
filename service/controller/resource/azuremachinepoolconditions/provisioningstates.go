package azuremachinepoolconditions

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

const (
	DeploymentNotFoundReason                 = "DeploymentNotFound"
	DeploymentProvisioningStateUnknownReason = "DeploymentProvisioningStateUnknown"
	DeploymentProvisioningStatePrefix        = "DeploymentProvisioningState"
	DeploymentProvisioningStateSucceeded     = "Succeeded"
	DeploymentProvisioningStateFailed        = "Failed"
)

func isProvisioningStateSet(deployment *resources.DeploymentExtended) bool {
	if deployment.Properties != nil &&
		deployment.Properties.ProvisioningState != nil &&
		*deployment.Properties.ProvisioningState != "" {
		return true
	}

	return false
}

func (r *Resource) setProvisioningStateWarningFailed(ctx context.Context, cr *capzexp.AzureMachinePool, deploymentName string, condition capi.ConditionType) {
	message := "Deployment %s failed, it might succeed after retrying, see Azure portal for more details"
	messageArgs := deploymentName
	reason := DeploymentProvisioningStatePrefix + DeploymentProvisioningStateFailed

	capiconditions.MarkFalse(
		cr,
		condition,
		reason,
		capi.ConditionSeverityError,
		message,
		messageArgs)

	r.logWarning(ctx, message, messageArgs)
}

func (r *Resource) setProvisioningStateWarning(ctx context.Context, cr *capzexp.AzureMachinePool, deploymentName string, currentProvisioningState string, condition capi.ConditionType) {
	message := "Deployment %s has not succeeded yet, current state is %s, " +
		"check back in few minutes, see Azure portal for more details"
	messageArgs := []interface{}{deploymentName, currentProvisioningState}
	reason := DeploymentProvisioningStatePrefix + currentProvisioningState

	capiconditions.MarkFalse(
		cr,
		condition,
		reason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logDebug(ctx, message, messageArgs...)
}

func (r *Resource) setProvisioningStateUnknown(ctx context.Context, cr *capzexp.AzureMachinePool, deploymentName string, condition capi.ConditionType) {
	message := "Deployment %s provisioning state not returned by Azure API, check back in few minutes"
	messageArgs := deploymentName
	capiconditions.MarkFalse(
		cr,
		condition,
		DeploymentProvisioningStateUnknownReason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logDebug(ctx, message, messageArgs)
}

func (r *Resource) setDeploymentNotFound(ctx context.Context, cr *capzexp.AzureMachinePool, deploymentName string, condition capi.ConditionType) {
	message := "Deployment %s is not found, check back in few minutes"
	messageArgs := deploymentName
	capiconditions.MarkFalse(
		cr,
		condition,
		DeploymentNotFoundReason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logDebug(ctx, message, messageArgs)
}
