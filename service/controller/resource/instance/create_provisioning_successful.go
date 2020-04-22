package instance

import (
	"context"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
)

func (r *Resource) provisioningSuccessfulTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "Worker VMSS deployment successfully provisioned")
	return ClusterUpgradeRequirementCheck, nil
}
