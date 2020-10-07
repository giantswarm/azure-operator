package nodepool

import (
	"context"
	"fmt"

	azureconditions "github.com/giantswarm/apiextensions/v2/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	conditions "sigs.k8s.io/cluster-api/util/conditions"
)

const (
	ProvisioningStateSucceeded = "Succeeded"
	ProvisioningStateFailed    = "Failed"
)

func (r *Resource) UpdateDeploymentSucceededCondition(ctx context.Context, azureMachinePool *capzexpv1alpha3.AzureMachinePool, provisioningState *string) error {
	conditionType := azureconditions.DeploymentSucceededCondition
	var conditionReason string
	var conditionSeverity capiv1alpha3.ConditionSeverity
	logger := r.Logger.With("level", "debug", "type", "AzureMachinePool", "message", "setting Status.Condition", "conditionType", conditionType)

	if provisioningState == nil {
		conditionReason = "DeploymentNotFound"
		conditionSeverity = capiv1alpha3.ConditionSeverityWarning
		conditions.MarkFalse(
			azureMachinePool,
			conditionType,
			conditionReason,
			conditionSeverity,
			"Deployment has not been found.")
		logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
	} else {
		switch *provisioningState {
		case ProvisioningStateSucceeded:
			conditions.MarkTrue(azureMachinePool, conditionType)
			logger.LogCtx(ctx, "conditionStatus", true)
		case ProvisioningStateFailed:
			conditionSeverity = capiv1alpha3.ConditionSeverityError
			conditionReason = "ProvisioningStateFailed"
			conditions.MarkFalse(
				azureMachinePool,
				conditionType,
				conditionReason,
				conditionSeverity,
				"Deployment has failed.")
			logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
		default:
			conditionSeverity = capiv1alpha3.ConditionSeverityWarning
			conditionReason = fmt.Sprintf("ProvisioningState%s", *provisioningState)
			conditions.MarkFalse(
				azureMachinePool,
				conditionType,
				conditionReason,
				conditionSeverity,
				"Current deployment provisioning status is %s.",
				*provisioningState)
			logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
		}
	}

	err := r.CtrlClient.Status().Update(ctx, azureMachinePool)
	if err != nil {
		r.Logger.LogCtx(ctx,
			"level", "error",
			"type", "AzureMachinePool",
			"conditionType", conditionType,
			"message", "error while setting Status.Condition")
		return microerror.Mask(err)
	}

	// in MachinePool: use conditions.SetSummary
	return nil
}
