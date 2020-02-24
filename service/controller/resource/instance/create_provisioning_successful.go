package instance

import (
	"context"

	state2 "github.com/giantswarm/azure-operator/service/controller/resource/instance/internal/state"
)

func (r *Resource) provisioningSuccessfulTransition(ctx context.Context, obj interface{}, currentState state2.State) (state2.State, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "VMSS deployment successfully provisioned")
	return ClusterUpgradeRequirementCheck, nil
}
