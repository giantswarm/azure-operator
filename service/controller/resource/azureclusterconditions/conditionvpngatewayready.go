package azureclusterconditions

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"

	azureconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	vpnDeploymentName = "vpn-template"

	DeploymentNotFoundReason                 = "DeploymentNotFound"
	DeploymentProvisioningStateUnknownReason = "DeploymentProvisioningStateUnknown"
	DeploymentProvisioningStatePrefix        = "DeploymentProvisioningState"
	DeploymentProvisioningStateSucceeded     = "Succeeded"
	DeploymentProvisioningStateFailed        = "Failed"
)

func (r *Resource) ensureVPNGatewayReadyCondition(ctx context.Context, azureCluster *capz.AzureCluster) error {
	r.logDebug(ctx, "ensuring condition %s", azureconditions.VPNGatewayReadyCondition)
	var err error

	// Get Azure Deployments client
	deploymentsClient, err := r.azureClientsFactory.GetDeploymentsClient(ctx, azureCluster.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	// Get VPN Gateway deployment
	deployment, err := deploymentsClient.Get(ctx, key.ClusterName(azureCluster), vpnDeploymentName)
	if IsNotFound(err) {
		// VPN Gateway deployment has not been found, which means that we still
		// didn't start deploying it.
		r.setVPNGatewayDeploymentNotFound(ctx, azureCluster)
		return nil
	} else if err != nil {
		// Error while getting VPN Gateway deployment, let's check if
		// deployment provisioning state is set.
		if !isProvisioningStateSet(&deployment) {
			return microerror.Mask(err)
		}

		currentProvisioningState := *deployment.Properties.ProvisioningState
		r.setProvisioningStateWarning(ctx, azureCluster, currentProvisioningState)
		return nil
	}

	// We got the VPN deployment without errors, but for some reason the provisioning state is
	// not set.
	if !isProvisioningStateSet(&deployment) {
		r.setProvisioningStateUnknown(ctx, azureCluster)
		return nil
	}

	// Now let's finally check what's the current VPN Gateway deployment
	// provisioning state.
	currentProvisioningState := *deployment.Properties.ProvisioningState

	switch currentProvisioningState {
	case DeploymentProvisioningStateSucceeded:
		// All good, VPN gateway deployment has been completed successfully! :)
		capiconditions.MarkTrue(azureCluster, azureconditions.VPNGatewayReadyCondition)
	case DeploymentProvisioningStateFailed:
		// VPN gateway deployment has failed.
		r.setProvisioningStateWarningFailed(ctx, azureCluster)
	default:
		// VPN gateway deployment is probably still running.
		r.setProvisioningStateWarning(ctx, azureCluster, currentProvisioningState)
	}

	r.logDebug(ctx, "finished ensuring condition %s", azureconditions.VPNGatewayReadyCondition)

	return nil
}

func isProvisioningStateSet(deployment *resources.DeploymentExtended) bool {
	if deployment.Properties != nil &&
		deployment.Properties.ProvisioningState != nil &&
		*deployment.Properties.ProvisioningState != "" {
		return true
	}

	return false
}

func (r *Resource) setProvisioningStateWarningFailed(ctx context.Context, azureCluster *capz.AzureCluster) {
	message := "VPN Gateway deployment %s failed, it might succeed after retrying, see Azure portal for more details"
	messageArgs := vpnDeploymentName
	reason := DeploymentProvisioningStatePrefix + DeploymentProvisioningStateFailed

	capiconditions.MarkFalse(
		azureCluster,
		azureconditions.VPNGatewayReadyCondition,
		reason,
		capi.ConditionSeverityError,
		message,
		messageArgs)

	r.logWarning(ctx, message, messageArgs)
}

func (r *Resource) setProvisioningStateWarning(ctx context.Context, azureCluster *capz.AzureCluster, currentProvisioningState string) {
	message := "VPN Gateway deployment %s has not succeeded yet, current state is %s, " +
		"check back in few minutes, see Azure portal for more details"
	messageArgs := []interface{}{vpnDeploymentName, currentProvisioningState}
	reason := DeploymentProvisioningStatePrefix + currentProvisioningState

	capiconditions.MarkFalse(
		azureCluster,
		azureconditions.VPNGatewayReadyCondition,
		reason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logDebug(ctx, message, messageArgs)
}

func (r *Resource) setProvisioningStateUnknown(ctx context.Context, azureCluster *capz.AzureCluster) {
	message := "VPN Gateway deployment %s provisioning state not returned by Azure API, check back in few minutes"
	messageArgs := vpnDeploymentName
	capiconditions.MarkFalse(
		azureCluster,
		azureconditions.VPNGatewayReadyCondition,
		DeploymentProvisioningStateUnknownReason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logDebug(ctx, message, messageArgs)
}

func (r *Resource) setVPNGatewayDeploymentNotFound(ctx context.Context, azureCluster *capz.AzureCluster) {
	message := "VPN Gateway deployment %s is not found, check back in few minutes"
	messageArgs := vpnDeploymentName
	capiconditions.MarkFalse(
		azureCluster,
		azureconditions.VPNGatewayReadyCondition,
		DeploymentNotFoundReason,
		capi.ConditionSeverityWarning,
		message,
		messageArgs)

	r.logDebug(ctx, message, messageArgs)
}
