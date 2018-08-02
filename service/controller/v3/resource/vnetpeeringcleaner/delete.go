package vnetpeeringcleaner

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"

	"github.com/giantswarm/azure-operator/client"
)

// NewDeletePatch is noop.
func (r *Resource) NewDeletePatch(ctx context.Context, azureConfig, current, desired interface{}) (*controller.Patch, error) {
	r.logger.Log("level", "debug", "message", "NewDeletePatch")
	return nil, nil
}

// newDeletePatch use desired as delete patch since it is mostly static and more likely to be present than current.
func (r *Resource) newDeletePatch(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired network.VirtualNetworkPeering) (*controller.Patch, error) {
	patch := controller.NewPatch()
	patch.SetDeleteChange(desired)
	return patch, nil
}

// ApplyDeleteChange is noop.
func (r *Resource) ApplyDeleteChange(ctx context.Context, azureConfig, change interface{}) error {
	r.logger.Log("level", "debug", "message", "ApplyDeleteChange")
	return nil
}

func (r *Resource) applyDeleteChange(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, change network.VirtualNetworkPeering) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "deleting host vnet peering")

	vnetPeeringClient, err := r.getVnetPeeringClient()
	if err != nil {
		return microerror.Mask(err)
	}

	respFuture, err := vnetPeeringClient.Delete(ctx, r.azure.HostCluster.ResourceGroup, r.azure.HostCluster.ResourceGroup, *change.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	res, err := vnetPeeringClient.DeleteResponder(respFuture.Response())
	if client.ResponseWasNotFound(res) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not find host vnet peering")
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleted host vnet peering")
	}

	return nil
}
