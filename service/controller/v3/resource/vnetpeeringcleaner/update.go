package vnetpeeringcleaner

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"

	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

// NewUpdatePatch provide a controller.Patch holding the needed network.VirtualNetworkPeering to be deleted.
func (r *Resource) NewUpdatePatch(ctx context.Context, azureConfig, current, desired interface{}) (*controller.Patch, error) {
	r.logger.Log("level", "debug", "message", "NewUpdatePatch")
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	c, err := toVnetPeering(current)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	d, err := toVnetPeering(desired)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch, err := r.newUpdatePatch(ctx, a, c, d)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Log("level", "debug", "message", "NewUpdatePatch", "patch", patch)
	return patch, nil
}

// newUpdatePatch use desired as patch since it is mostly static and more likely to be present than current.
func (r *Resource) newUpdatePatch(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired network.VirtualNetworkPeering) (*controller.Patch, error) {
	patch := controller.NewPatch()
	patch.SetUpdateChange(desired)
	return patch, nil
}

// ApplyUpdateChange perform the host cluster virtual network peering delete against azure.
func (r *Resource) ApplyUpdateChange(ctx context.Context, azureConfig, change interface{}) error {
	r.logger.Log("level", "debug", "message", "ApplyUpdateChange")
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return microerror.Mask(err)
	}
	c, err := toVnetPeering(change)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.applyDeleteChange(ctx, a, c)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensure host vnet peering: deleted")
	return nil
}
