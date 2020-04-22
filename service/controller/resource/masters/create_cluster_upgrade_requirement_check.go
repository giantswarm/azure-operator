package masters

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/pkg/helpers"
	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/resource/internal/state"
)

func (r *Resource) clusterUpgradeRequirementCheckTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	isCreating := helpers.IsClusterCreating(cr)
	anyOldNodes, err := helpers.AnyNodesOutOfDate(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if !isCreating && anyOldNodes {
		// Only continue rolling nodes when cluster is not creating and there
		// are old nodes in tenant cluster.
		return MasterInstancesUpgrading, nil
	}

	// Skip instance rolling by default.
	return DeploymentCompleted, nil
}
