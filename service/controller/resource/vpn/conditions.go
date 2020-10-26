package vpn

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	ProvisioningStateSucceeded = "Succeeded"
	ProvisioningStateFailed    = "Failed"
)

func (r *Resource) UpdateVPNGatewayReadyCondition(ctx context.Context, azureConfig v1alpha1.AzureConfig, provisioningState *string) error {
	conditionType := azure.VPNGatewayReadyCondition
	var conditionReason string
	var conditionSeverity v1alpha3.ConditionSeverity
	logger := r.logger.With("level", "debug", "type", "AzureCluster", "message", "setting Status.Condition", "conditionType", conditionType)

	// Update AzureCluster conditions
	organizationNamespace := key.OrganizationNamespace(&azureConfig)
	azureCluster, err := helpers.GetAzureClusterByName(ctx, r.ctrlClient, organizationNamespace, azureConfig.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	if provisioningState == nil {
		conditionReason = "DeploymentNotFound"
		conditionSeverity = v1alpha3.ConditionSeverityWarning
		conditions.MarkFalse(
			azureCluster,
			conditionType,
			conditionReason,
			conditionSeverity,
			"Deployment has not been found.")
		logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
	} else {
		switch *provisioningState {
		case ProvisioningStateSucceeded:
			conditions.MarkTrue(azureCluster, conditionType)
			logger.LogCtx(ctx, "conditionStatus", true)
		case ProvisioningStateFailed:
			conditionSeverity = v1alpha3.ConditionSeverityError
			conditionReason = "ProvisioningStateFailed"
			conditions.MarkFalse(
				azureCluster,
				conditionType,
				conditionReason,
				conditionSeverity,
				"Deployment has failed.")
			logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
		default:
			conditionSeverity = v1alpha3.ConditionSeverityWarning
			conditionReason = fmt.Sprintf("ProvisioningState%s", *provisioningState)
			conditions.MarkFalse(
				azureCluster,
				conditionType,
				conditionReason,
				conditionSeverity,
				"Current deployment provisioning status is %s.",
				*provisioningState)
			logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
		}
	}

	// Note: Updating of AzureCluster conditions should not be done here synchronously, but
	// probably in a separate handler. This is an alpha implementation.
	err = r.ctrlClient.Status().Update(ctx, azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
