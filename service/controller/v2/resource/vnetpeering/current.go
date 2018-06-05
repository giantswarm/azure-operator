package vnetpeering

import (
	"context"
	"fmt"

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

	// In order to make vnet peering work we need a virtual network which we can
	// use to peer. In case there is no virtual network yet we cancel the resource
	// and try again on the next resync period. This is a classical scenario on
	// guest cluster creation. If we would not check for the virtual network
	// existence the client calls of CreateOrUpdate would fail with not found
	// errors.
	if !key.IsDeleted(customObject) {
		c, err := r.getVirtualNetworksClient(ctx)
		if err != nil {
			return network.VirtualNetworkPeering{}, microerror.Mask(err)
		}

		g := key.ResourceGroupName(customObject)
		n := key.VnetName(customObject)
		e := ""
		v, err := c.Get(ctx, g, n, e)
		if IsVirtualNetworkNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the virtual network in the Azure API")
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return network.VirtualNetworkPeering{}, nil
		} else if err != nil {
			return network.VirtualNetworkPeering{}, microerror.Mask(err)
		} else {
			s := *v.ProvisioningState

			if !key.IsFinalProvisioningState(s) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("virtual network is in state '%s'", s))
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

				return network.VirtualNetworkPeering{}, nil
			}
		}
	}

	// Look for the current state of the vnet peering. It is a valid operation to
	// not find any state. This indicates we want to create the vnet peering in
	// the following steps.
	var vnetPeering network.VirtualNetworkPeering
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the vnet peerings in the Azure API")

		c, err := r.getVnetPeeringClient(ctx)
		if err != nil {
			return network.VirtualNetworkPeering{}, microerror.Mask(err)
		}

		g := r.azure.HostCluster.ResourceGroup
		n := key.ResourceGroupName(customObject)
		vnetPeering, err = c.Get(ctx, g, g, n)
		if client.ResponseWasNotFound(vnetPeering.Response) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the vnet peerings in the Azure API")

			return network.VirtualNetworkPeering{}, nil
		} else if err != nil {
			return network.VirtualNetworkPeering{}, microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "found the vnet peerings in the Azure API")
	}

	return vnetPeering, nil
}
