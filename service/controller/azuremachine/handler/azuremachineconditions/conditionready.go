package azuremachineconditions

import (
	"context"

	azureconditions "github.com/giantswarm/apiextensions/v6/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

func (r *Resource) ensureReadyCondition(ctx context.Context, azureMachine *capz.AzureMachine) error {
	r.logger.Debugf(ctx, "ensuring condition Ready")
	var err error

	// Ensure SubnetReady condition
	err = r.ensureSubnetReadyCondition(ctx, azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	// Ensure VMMSReady condition
	err = r.ensureVMSSReadyCondition(ctx, azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	// List of conditions that all need to be True for the Ready condition to
	// be True:
	// - SubnetReady
	// - VMSSReady
	conditionsToSummarize := capiconditions.WithConditions(
		azureconditions.SubnetReadyCondition,
		azureconditions.VMSSReadyCondition)

	// Update Ready condition
	capiconditions.SetSummary(
		azureMachine,
		conditionsToSummarize,
		capiconditions.AddSourceRef())

	// Now check current Ready condition so we can log the value
	r.logConditionStatus(ctx, azureMachine, capi.ReadyCondition)
	r.logger.Debugf(ctx, "ensured condition Ready")

	azureMachine.Status.Ready = capiconditions.IsTrue(azureMachine, capi.ReadyCondition)

	return nil
}
