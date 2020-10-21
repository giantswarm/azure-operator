package clusterconditions

import (
	"context"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

func (r *Resource) ensureReadyCondition(ctx context.Context, cluster *capi.Cluster) error {
	var err error

	err = r.ensureProviderInfrastructureReadyCondition(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureNodePoolsReadyCondition(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	// List of conditions that all need to be True for the Ready condition to be True
	conditionsToSummarize := capiconditions.WithConditions(
		aeconditions.ProviderInfrastructureReadyCondition,
		aeconditions.NodePoolsReadyCondition)

	capiconditions.SetSummary(
		cluster,
		conditionsToSummarize,
		capiconditions.AddSourceRef())

	return nil
}
