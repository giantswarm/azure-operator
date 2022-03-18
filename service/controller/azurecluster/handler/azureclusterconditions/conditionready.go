package azureclusterconditions

import (
	"context"

	azureconditions "github.com/giantswarm/apiextensions/v5/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

func (r *Resource) ensureReadyCondition(ctx context.Context, azureCluster *capz.AzureCluster) error {
	r.logger.Debugf(ctx, "setting condition Ready")

	// Note: This is an incomplete implementation that checks only resource
	// group, because it's created in the beginning, and the VPN Gateway,
	// because it's created at the end. Final implementation should include
	// checking of other Azure resources as well. and it will be done in
	// AzureCluster controller.
	err := r.ensureVNetPeeringReadyCondition(ctx, azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	// List of conditions that all need to be True for the Ready condition to
	// be True.
	conditionsToSummarize := capiconditions.WithConditions(
		azureconditions.ResourceGroupReadyCondition,
		VNetPeeringReadyCondition)

	capiconditions.SetSummary(
		azureCluster,
		conditionsToSummarize,
		capiconditions.AddSourceRef())

	readyCondition := capiconditions.Get(azureCluster, capi.ReadyCondition)

	if readyCondition == nil {
		r.logger.Debugf(ctx, "condition Ready not set")
	} else {
		messageFormat := "condition Ready set to %s"
		messageArgs := []interface{}{readyCondition.Status}
		if readyCondition.Status != corev1.ConditionTrue {
			messageFormat += ", Reason=%s, Severity=%s, Message=%s"
			messageArgs = append(messageArgs, readyCondition.Reason)
			messageArgs = append(messageArgs, readyCondition.Severity)
			messageArgs = append(messageArgs, readyCondition.Message)
		}
		r.logger.Debugf(ctx, messageFormat, messageArgs...)

		azureCluster.Status.Ready = capiconditions.IsTrue(azureCluster, capi.ReadyCondition)
	}

	return nil
}
