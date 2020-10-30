package azuremachinepoolconditions

import (
	"context"

	azureconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

func (r *Resource) ensureReadyCondition(ctx context.Context, azureMachinePool *capzexp.AzureMachinePool) error {
	r.logDebug(ctx, "ensuring condition Ready")
	var err error

	// Ensure VMMSReady condition
	err = r.ensureSubnetReadyCondition(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	// Ensure VMMSReady condition
	err = r.ensureVmssReadyCondition(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	// List of conditions that all need to be True for the Ready condition to
	// be True:
	// - VMSSReady: node pool VMSS is ready
	// - SubnetReady: node pool subnet is ready
	conditionsToSummarize := capiconditions.WithConditions(
		azureconditions.SubnetReadyCondition,
		azureconditions.VMSSReadyCondition)

	// Update Ready condition
	capiconditions.SetSummary(
		azureMachinePool,
		conditionsToSummarize,
		capiconditions.AddSourceRef())

	// Now check current Ready condition so we can log the value
	r.logConditionStatus(ctx, azureMachinePool, capi.ReadyCondition)
	r.logDebug(ctx, "ensured condition Ready")
	return nil
}

func (r *Resource) logConditionStatus(ctx context.Context, azureMachinePool *capzexp.AzureMachinePool, conditionType capi.ConditionType) {
	condition := capiconditions.Get(azureMachinePool, conditionType)

	if condition == nil {
		r.logWarning(ctx, "condition %s not set", conditionType)
	} else {
		messageFormat := "condition %s set to %s"
		messageArgs := []interface{}{conditionType, condition.Status}
		if condition.Status != corev1.ConditionTrue {
			messageFormat += ", Reason=%s, Severity=%s, Message=%s"
			messageArgs = append(messageArgs, condition.Reason)
			messageArgs = append(messageArgs, condition.Severity)
			messageArgs = append(messageArgs, condition.Message)
		}
		r.logDebug(ctx, messageFormat, messageArgs...)
	}
}
