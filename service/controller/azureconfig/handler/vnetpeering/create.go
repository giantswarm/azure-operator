package vnetpeering

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/reconciliationcanceledcontext"
	"github.com/giantswarm/to"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	ProvisioningStateDeleting = "Deleting"
)

// This resource manages the VNet peering between the control plane and tenant cluster.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var tcVnet network.VirtualNetwork
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Checking if TC virtual network %#q exists in resource group %#q", key.VnetName(cr), key.ResourceGroupName(cr)))

		virtualNetworksClient, err := r.clientFactory.GetVirtualNetworksClient(ctx, cr.ObjectMeta)
		if err != nil {
			return microerror.Mask(err)
		}

		tcVnet, err = virtualNetworksClient.Get(ctx, key.ResourceGroupName(cr), key.VnetName(cr), "")
		if IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "TC Virtual network does not exist in resource group")
			reconciliationcanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "TC Virtual network exists in resource group")
	}

	var cpVnet network.VirtualNetwork
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Checking if CP virtual network %#q exists in resource group %#q", r.mcVirtualNetworkName, r.mcResourceGroup))

		cpVnet, err = r.cpAzureClientSet.VirtualNetworkClient.Get(ctx, r.mcResourceGroup, r.mcVirtualNetworkName, "")
		if IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "CP Virtual network does not exist in resource group")
			reconciliationcanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "CP Virtual network exists")
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Ensuring vnet peering %#q exists on the tenant cluster vnet %#q in resource group %#q", r.mcVirtualNetworkName, key.VnetName(cr), key.ResourceGroupName(cr)))

		vnetPeeringsClient, err := r.clientFactory.GetVnetPeeringsClient(ctx, cr.ObjectMeta)
		if err != nil {
			return microerror.Mask(err)
		}

		tcPeering := r.getTCVnetPeering(*cpVnet.ID)
		_, err = vnetPeeringsClient.CreateOrUpdate(ctx, key.ResourceGroupName(cr), key.VnetName(cr), r.mcVirtualNetworkName, tcPeering)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Ensuring vnet peering %#q exists on the control plane vnet %#q in resource group %#q", key.ResourceGroupName(cr), r.mcVirtualNetworkName, r.mcResourceGroup))
		cpPeering := r.getCPVnetPeering(*tcVnet.ID)
		_, err = r.cpAzureClientSet.VnetPeeringClient.CreateOrUpdate(ctx, r.mcResourceGroup, r.mcVirtualNetworkName, key.ResourceGroupName(cr), cpPeering)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) getCPVnetPeering(vnetId string) network.VirtualNetworkPeering {
	peering := network.VirtualNetworkPeering{
		VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
			AllowVirtualNetworkAccess: to.BoolP(true),
			AllowForwardedTraffic:     to.BoolP(false),
			AllowGatewayTransit:       to.BoolP(false),
			UseRemoteGateways:         to.BoolP(false),
			RemoteVirtualNetwork: &network.SubResource{
				ID: &vnetId,
			},
		},
	}

	return peering
}

func (r *Resource) getTCVnetPeering(vnetId string) network.VirtualNetworkPeering {
	peering := network.VirtualNetworkPeering{
		VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
			AllowVirtualNetworkAccess: to.BoolP(true),
			AllowForwardedTraffic:     to.BoolP(false),
			AllowGatewayTransit:       to.BoolP(false),
			UseRemoteGateways:         to.BoolP(false),
			RemoteVirtualNetwork: &network.SubResource{
				ID: &vnetId,
			},
		},
	}

	return peering
}
