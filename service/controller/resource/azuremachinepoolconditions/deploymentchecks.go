package azuremachinepoolconditions

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	DeploymentNotFoundReason                 = "DeploymentNotFound"
	DeploymentProvisioningStateUnknownReason = "DeploymentProvisioningStateUnknown"
	DeploymentProvisioningStatePrefix        = "DeploymentProvisioningState"
	DeploymentProvisioningStateSucceeded     = "Succeeded"
	DeploymentProvisioningStateFailed        = "Failed"
)

func (r *Resource) checkIfDeploymentIsSuccessful(ctx context.Context, deploymentsClient *resources.DeploymentsClient, cr capiconditions.Setter, deploymentName string, conditionType capi.ConditionType) (bool, error) {
	deployment, err := deploymentsClient.Get(ctx, key.ClusterName(cr), deploymentName)
	if IsNotFound(err) {
		// Deployment has not been found, which means that we still
		// didn't start deploying it.
		r.setDeploymentNotFound(ctx, cr, deploymentName, conditionType)
		return false, nil
	} else if err != nil {
		// Error while getting Subnet deployment, let's check if
		// deployment provisioning state is set.
		if !isProvisioningStateSet(&deployment) {
			return false, microerror.Mask(err)
		}

		currentProvisioningState := *deployment.Properties.ProvisioningState
		r.setProvisioningStateWarning(ctx, cr, deploymentName, currentProvisioningState, conditionType)
		return false, nil
	}

	// We got the Subnet deployment without errors, but for some reason the provisioning state is
	// not set.
	if !isProvisioningStateSet(&deployment) {
		r.setProvisioningStateUnknown(ctx, cr, deploymentName, conditionType)
		return false, nil
	}

	// Now let's finally check what's the current subnet deployment
	// provisioning state.
	currentProvisioningState := *deployment.Properties.ProvisioningState

	switch currentProvisioningState {
	case DeploymentProvisioningStateSucceeded:
		return true, nil
	case DeploymentProvisioningStateFailed:
		// Subnet deployment has failed.
		r.setProvisioningStateWarningFailed(ctx, cr, deploymentName, conditionType)
	default:
		// Subnet deployment is probably still running.
		r.setProvisioningStateWarning(ctx, cr, deploymentName, currentProvisioningState, conditionType)
	}

	return false, nil
}

func isProvisioningStateSet(deployment *resources.DeploymentExtended) bool {
	if deployment.Properties != nil &&
		deployment.Properties.ProvisioningState != nil &&
		*deployment.Properties.ProvisioningState != "" {
		return true
	}

	return false
}

func (r *Resource) setProvisioningStateWarningFailed(ctx context.Context, cr capiconditions.Setter, deploymentName string, condition capi.ConditionType) {
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

func (r *Resource) setProvisioningStateWarning(ctx context.Context, cr capiconditions.Setter, deploymentName string, currentProvisioningState string, condition capi.ConditionType) {
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
		messageArgs...)

	r.logWarning(ctx, message, messageArgs...)
}

func (r *Resource) setProvisioningStateUnknown(ctx context.Context, cr capiconditions.Setter, deploymentName string, condition capi.ConditionType) {
	message := "Deployment %s provisioning state not returned by Azure API, check back in few minutes"
	messageArgs := deploymentName
	capiconditions.MarkFalse(
		cr,
		condition,
		DeploymentProvisioningStateUnknownReason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logWarning(ctx, message, messageArgs)
}

func (r *Resource) setDeploymentNotFound(ctx context.Context, cr capiconditions.Setter, deploymentName string, condition capi.ConditionType) {
	message := "Deployment %s is not found, check back in few minutes"
	messageArgs := deploymentName
	capiconditions.MarkFalse(
		cr,
		condition,
		DeploymentNotFoundReason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logWarning(ctx, message, messageArgs)
}
