package masters

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/pkg/helpers"
	"github.com/giantswarm/azure-operator/service/controller/resource/internal/state"
)

func (r *Resource) waitForMastersToBecomeReadyTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if all tenant cluster master nodes are Ready")

	readyForTransitioning, err := helpers.AreMasterNodesReadyForTransitioning(ctx)
	if helpers.IsClientNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if !readyForTransitioning {
		r.logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster master nodes are not Ready")
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster master nodes are Ready")

	return DeploymentCompleted, nil
}
