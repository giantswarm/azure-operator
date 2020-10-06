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

	if provisioningState == nil {
		conditions.MarkFalse(
			azureMachinePool,
			conditionType,
			"DeploymentNotFound",
			capiv1alpha3.ConditionSeverityWarning,
			"Deployment has not been found.")
	} else {
		switch *provisioningState {
		case ProvisioningStateSucceeded:
			conditions.MarkTrue(azureMachinePool, conditionType)
		case ProvisioningStateFailed:
			conditions.MarkFalse(
				azureMachinePool,
				conditionType,
				"ProvisioningStateFailed",
				capiv1alpha3.ConditionSeverityError,
				"Deployment has failed.")
		default:
			conditions.MarkFalse(
				azureMachinePool,
				conditionType,
				fmt.Sprintf("ProvisioningState%s", *provisioningState),
				capiv1alpha3.ConditionSeverityError,
				"Current deployment provisioning status is %s.",
				*provisioningState)
		}
	}

	err := r.CtrlClient.Status().Update(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	// in MachinePool: use conditions.SetSummary
	return nil
}
