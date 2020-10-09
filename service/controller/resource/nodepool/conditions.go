package nodepool

import (
	"context"
	"fmt"

	azureconditions "github.com/giantswarm/apiextensions/v2/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	conditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v4/pkg/helpers"
)

const (
	ProvisioningStateSucceeded = "Succeeded"
	ProvisioningStateFailed    = "Failed"
)

func (r *Resource) UpdateDeploymentSucceededCondition(ctx context.Context, azureMachinePool *capzexp.AzureMachinePool, provisioningState *string) error {
	conditionType := azureconditions.DeploymentSucceededCondition
	var conditionReason string
	var conditionSeverity capi.ConditionSeverity
	logger := r.Logger.With("level", "debug", "type", "AzureMachinePool", "message", "setting Status.Condition", "conditionType", conditionType)

	if provisioningState == nil {
		conditionReason = "DeploymentNotFound"
		conditionSeverity = capi.ConditionSeverityWarning
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
			conditionSeverity = capi.ConditionSeverityError
			conditionReason = "ProvisioningStateFailed"
			conditions.MarkFalse(
				azureMachinePool,
				conditionType,
				conditionReason,
				conditionSeverity,
				"Deployment has failed.")
			logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
		default:
			conditionSeverity = capi.ConditionSeverityWarning
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

	// Preview implementation only: DeploymentSucceeded -> Ready
	// In the final version it will include more detailed and more accurate conditions, e.g. checking the power state of VMSS instances.
	if conditions.IsTrue(azureMachinePool, azureconditions.DeploymentSucceededCondition) {
		conditions.MarkTrue(azureMachinePool, capi.ReadyCondition)
	} else {
		conditionReason = "Deploying"
		conditionSeverity = capi.ConditionSeverityWarning
		conditions.MarkFalse(
			azureMachinePool,
			capi.ReadyCondition,
			conditionReason,
			conditionSeverity,
			"Node pool deployment is in progress.")
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

	// Note: Updating of AzureCluster conditions should not be done here synchronously, but
	// probably in a separate handler. This is an alpha implementation.

	// Update AzureCluster conditions
	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}
	err = helpers.UpdateAzureClusterConditions(ctx, r.CtrlClient, r.Logger, azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	// in MachinePool: use conditions.SetSummary
	return nil
}
