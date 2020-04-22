package instance

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/pkg/helpers"
	"github.com/giantswarm/azure-operator/service/controller/resource/internal/state"
)

func (r *Resource) waitForWorkersToBecomeReadyTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if all tenant cluster worker nodes are Ready")

	readyForTransitioning, err := helpers.AreWorkerNodesReadyForTransitioning(ctx)
	if helpers.IsClientNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if !readyForTransitioning {
		r.logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster worker nodes are not Ready")
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster worker nodes are Ready")

	return DrainOldWorkerNodes, nil
}
