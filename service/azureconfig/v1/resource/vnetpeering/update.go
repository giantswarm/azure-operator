package vnetpeering

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/framework"
)

func (r Resource) NewUpdatePatch(ctx context.Context, azureConfig, current, desired interface{}) (*framework.Patch, error) {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}
	c, err := toVnetPeering(current)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}
	d, err := toVnetPeering(desired)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}

	return r.newUpdatePatch(ctx, a, c, d)
}

func (r Resource) newUpdatePatch(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired network.VirtualNetworkPeering) (*framework.Patch, error) {
	patch := framework.NewPatch()

	if needUpdate(current, desired) {
		patch.SetUpdateChange(desired)
	}

	return patch, nil
}

func needUpdate(current, desired network.VirtualNetworkPeering) bool {
	if current.Name == nil || *current.Name != *desired.Name {
		return true
	}

	if current.VirtualNetworkPeeringPropertiesFormat == nil {
		return true
	}

	if current.VirtualNetworkPeeringPropertiesFormat.AllowVirtualNetworkAccess == nil ||
		*current.VirtualNetworkPeeringPropertiesFormat.AllowVirtualNetworkAccess != *desired.VirtualNetworkPeeringPropertiesFormat.AllowVirtualNetworkAccess {
		return true
	}

	if current.VirtualNetworkPeeringPropertiesFormat.RemoteVirtualNetwork == nil ||
		current.VirtualNetworkPeeringPropertiesFormat.RemoteVirtualNetwork.ID == nil ||
		*current.VirtualNetworkPeeringPropertiesFormat.RemoteVirtualNetwork.ID != *desired.VirtualNetworkPeeringPropertiesFormat.RemoteVirtualNetwork.ID {
		return true
	}

	return false
}

// ApplyUpdateChange perform the host cluster virtual network peering update against azure.
func (r Resource) ApplyUpdateChange(ctx context.Context, azureConfig, change interface{}) error {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return microerror.Maskf(err, "ApplyUpdateChange")
	}
	c, err := toVnetPeering(change)
	if err != nil {
		return microerror.Maskf(err, "ApplyUpdateChange")
	}

	return r.applyUpdateChange(ctx, a, c)
}

func (r *Resource) applyUpdateChange(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, change network.VirtualNetworkPeering) error {
	r.logger.LogCtx(ctx, "debug", "ensure host vnet peering")

	vnetPeeringClient, err := r.getVnetPeeringClient()
	if err != nil {
		return microerror.Maskf(err, "ensure host vnet peering")
	}

	// TODO: use host virtual network name instead of host resource group
	_, err = vnetPeeringClient.CreateOrUpdate(ctx, key.HostClusterResourceGroup(azureConfig), key.HostClusterResourceGroup(azureConfig), key.ResourceGroupName(azureConfig), change)
	if err != nil {
		return microerror.Maskf(err, "ensure host vnet peering %#v", change)
	}

	r.logger.LogCtx(ctx, "debug", "ensure host vnet peering: created")
	return nil
}
