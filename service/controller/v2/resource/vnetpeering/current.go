package vnetpeering

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

// GetCurrentState retrieve the current host cluster virtual network peering
// resource from azure.
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return network.VirtualNetworkPeering{}, microerror.Mask(err)
	}

	{
		c, err := r.getVirtualNetworksClient()
		if err != nil {
			return network.VirtualNetworkPeering{}, microerror.Mask(err)
		}

		g := r.azure.HostCluster.ResourceGroup
		n := r.azure.HostCluster.VirtualNetwork
		e := ""
		_, err = c.Get(ctx, g, n, e)
		if IsVirtualNetworkNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the Virtual Network in the Azure API")
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return network.VirtualNetworkPeering{}, nil
		} else if err != nil {
			return network.VirtualNetworkPeering{}, microerror.Mask(err)
		}
	}

	var vnetPeering network.VirtualNetworkPeering
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the Vnet Peerings in the Azure API")

		c, err := r.getVnetPeeringClient()
		if err != nil {
			return network.VirtualNetworkPeering{}, microerror.Mask(err)
		}

		g := r.azure.HostCluster.ResourceGroup
		n := key.ResourceGroupName(customObject)
		vnetPeering, err = c.Get(ctx, g, g, n)
		if client.ResponseWasNotFound(vnetPeering.Response) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the Vnet Peerings in the Azure API")

			return network.VirtualNetworkPeering{}, nil
		} else if err != nil {
			return network.VirtualNetworkPeering{}, microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "found the Vnet Peerings in the Azure API")
	}

	return vnetPeering, nil
}
