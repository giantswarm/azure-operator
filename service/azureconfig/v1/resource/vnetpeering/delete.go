package vnetpeering

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
)

// NewDeletePatch provide a framework.Patch holding the network.VirtualNetworkPeering to be deleted.
func (r Resource) NewDeletePatch(ctx context.Context, azureConfig, current, desired interface{}) (*framework.Patch, error) {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}
	c, err := toVnetPeering(current)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}
	d, err := toVnetPeering(desired)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}

	patch, err := r.newDeletePatch(ctx, a, c, d)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}

	return patch, nil
}

// newDeletePatch use desired as delete patch since it is mostly static and more likely to be present than current.
func (r Resource) newDeletePatch(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired network.VirtualNetworkPeering) (*framework.Patch, error) {
	patch := framework.NewPatch()
	patch.SetDeleteChange(desired)
	return patch, nil
}

// ApplyDeleteChange perform deletion of the change virtual network peering against azure.
func (r Resource) ApplyDeleteChange(ctx context.Context, azureConfig, change interface{}) error {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return microerror.Maskf(err, "ApplyDeleteChange")
	}
	c, err := toVnetPeering(change)
	if err != nil {
		return microerror.Maskf(err, "ApplyDeleteChange")
	}

	err = r.applyDeleteChange(ctx, a, c)
	if err != nil {
		return microerror.Maskf(err, "ApplyDeleteChange")
	}

	return nil
}

func (r Resource) applyDeleteChange(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, change network.VirtualNetworkPeering) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "deleting host vnet peering")

	vnetPeeringClient, err := r.getVnetPeeringClient()
	if err != nil {
		return microerror.Maskf(err, "deleting host vnet peering")
	}

	respFuture, err := vnetPeeringClient.Delete(ctx, key.HostClusterResourceGroup(azureConfig), key.HostClusterResourceGroup(azureConfig), *change.Name)
	if err != nil {
		return microerror.Maskf(err, "deleting host vnet peering")
	}

	res, err := vnetPeeringClient.DeleteResponder(respFuture.Response())
	if client.ResponseWasNotFound(res) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting host vnet peering: already deleted")
		return nil
	}
	if err != nil {
		return microerror.Maskf(err, "deleting host vnet peering")
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "deleting host vnet peering: deleted")
	return nil
}
