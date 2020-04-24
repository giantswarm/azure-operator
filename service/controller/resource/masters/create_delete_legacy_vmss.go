package masters

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) deleteLegacyVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	// Delete the scaleset
	err = r.deleteScaleSet(ctx, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
	if IsScaleSetNotFound(err) {
		// Scale set not found, all good.
		return DeploymentCompleted, nil
	} else if err != nil {
		return Empty, microerror.Mask(err)
	}

	return DeploymentCompleted, nil
}

func (r *Resource) deleteScaleSet(ctx context.Context, resourceGroup string, vmssName string) error {
	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.Delete(ctx, resourceGroup, vmssName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
