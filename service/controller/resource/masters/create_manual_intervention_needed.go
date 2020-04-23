package masters

import (
	"context"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
)

// This transition function aims at detecting if the master VMSS needs to be migrated from CoreOS to flatcar.
func (r *Resource) manualInterventionRequiredTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "error", "message", "The reconciliation on the masters resource can't continute. Manual intervention needed.")
	return currentState, nil
}
