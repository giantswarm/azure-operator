package machinepoolconditions

import (
	"context"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

func (r *Resource) ensureReadyCondition(ctx context.Context, machinePool *capiexp.MachinePool) error {
	r.logDebug(ctx, "ensuring condition Ready")

	// Ensure ProviderInfrastructureReady conditions
	err := r.ensureProviderInfrastructureReadyCondition(ctx, machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	// List of conditions that all need to be True for the Ready condition to
	// be True.
	// Currently we only check ProviderInfrastructureReady which mirrors
	// AzureMachinePool Ready, but we should also include checking of Node CRs
	// Ready.
	conditionsToSummarize := capiconditions.WithConditions(
		aeconditions.ProviderInfrastructureReadyCondition)

	// Update Ready condition
	capiconditions.SetSummary(
		machinePool,
		conditionsToSummarize,
		capiconditions.AddSourceRef())

	// Log condition change
	readyCondition := capiconditions.Get(machinePool, capi.ReadyCondition)

	if readyCondition == nil {
		r.logWarning(ctx, "condition Ready not set")
	} else {
		messageFormat := "condition Ready set to %s"
		messageArgs := []interface{}{readyCondition.Status}
		if readyCondition.Status != corev1.ConditionTrue {
			messageFormat += ", Reason=%s, Severity=%s, Message=%s"
			messageArgs = append(messageArgs, readyCondition.Reason)
			messageArgs = append(messageArgs, readyCondition.Severity)
			messageArgs = append(messageArgs, readyCondition.Message)
		}
		r.logDebug(ctx, messageFormat, messageArgs...)
	}

	r.logDebug(ctx, "ensured condition Ready")
	return nil
}
