package masters

import (
	"context"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
)

func (r *Resource) manualInterventionRequiredTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.Logger.LogCtx(ctx, "level", "error", "message", "The reconciliation on the masters resource can't continute. Manual intervention needed.")
	return currentState, nil
}
