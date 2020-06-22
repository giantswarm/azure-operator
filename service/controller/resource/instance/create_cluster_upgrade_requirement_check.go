package instance

import (
	"context"

	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes"
)

func (r *Resource) clusterUpgradeRequirementCheckTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	isCreating := *azureMachinePool.Status.ProvisioningState == v1alpha3.VMStateCreating
	anyOldNodes, err := nodes.AnyOutOfDate(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if !isCreating && anyOldNodes {
		// Only continue rolling nodes when cluster is not creating and there
		// are old nodes in tenant cluster.
		return ScaleUpWorkerVMSS, nil
	}

	// Skip instance rolling by default.
	return DeploymentCompleted, nil
}
