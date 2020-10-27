package azuremachinepoolconditions

import (
	"context"

	azureconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions/azure"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
)

func (r *Resource) ensureVmssReadyCondition(ctx context.Context, azureMachinePool *capzexp.AzureMachinePool) error {
	r.logDebug(ctx, "ensuring condition %s", azureconditions.VMSSReadyCondition)

	r.logConditionStatus(ctx, azureMachinePool, azureconditions.VMSSReadyCondition)
	r.logDebug(ctx, "ensured condition %s", azureconditions.VMSSReadyCondition)
	return nil
}
