package masters

import (
	"context"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) deallocateLegacyInstanceTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	deallocated, err := r.isLegacyVMSSInstanceDeallocated(ctx, cr)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	if !deallocated {
		r.logger.LogCtx(ctx, "level", "info", "message", "Legacy VMSS instance is not deallocated yet.")
		r.logger.LogCtx(ctx, "level", "info", "message", "Deallocating legacy VMSS instances.")
		err := r.deallocateLegacyInstances(ctx, key.LegacyMasterVMSSName(cr))
		if err != nil {
			return Empty, microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", "Deallocated legacy VMSS instances.")
		return currentState, nil
	}

	return DeploymentUninitialized, nil
}

func (r *Resource) deallocateLegacyInstances(ctx context.Context, vmssName string) error {
	// TODO deallocate all instances in the legacy VMSS
	return nil
}

func (r *Resource) isLegacyVMSSInstanceDeallocated(ctx context.Context, cr v1alpha1.AzureConfig) (bool, error) {
	// TODO check if all the legacy VMSS instances are deallocated
	return false, nil
}
