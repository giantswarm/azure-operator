package masters

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes"
)

func (r *Resource) clusterUpgradeRequirementCheckTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	isCreating := nodes.IsClusterCreating(cr)
	anyOldNodes, err := nodes.AnyNodesOutOfDate(ctx)
	if IsClientNotFound(err) {
		// The kubernetes API is down.
		// We check if the Legacy Master VMSS exists and in that case
		// we assume this is because we're migrating to Flatcar.
		exists, err := r.vmssExists(ctx, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
		if err != nil || !exists {
			return "", microerror.Mask(err)
		}
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if !isCreating && anyOldNodes {
		// Only continue rolling nodes when cluster is not creating and there
		// are old nodes in tenant cluster.
		return MasterInstancesUpgrading, nil
	}

	// Skip instance rolling by default.
	return WaitForRestore, nil
}
