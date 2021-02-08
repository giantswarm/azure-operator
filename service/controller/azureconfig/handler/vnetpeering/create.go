package vnetpeering

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
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
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Checking if CP virtual network %#q exists in resource group %#q", r.hostVirtualNetworkName, r.hostResourceGroup))

		cpVnet, err = r.cpAzureClientSet.VirtualNetworkClient.Get(ctx, r.hostResourceGroup, r.hostVirtualNetworkName, "")
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
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Ensuring vnet peering %#q exists on the tenant cluster vnet %#q in resource group %#q", r.hostVirtualNetworkName, key.VnetName(cr), key.ResourceGroupName(cr)))

		vnetPeeringsClient, err := r.clientFactory.GetVnetPeeringsClient(ctx, cr.ObjectMeta)
		if err != nil {
			return microerror.Mask(err)
		}

		tcPeering := r.getTCVnetPeering(*cpVnet.ID)
		_, err = vnetPeeringsClient.CreateOrUpdate(ctx, key.ResourceGroupName(cr), key.VnetName(cr), r.hostVirtualNetworkName, tcPeering)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Ensuring vnet peering %#q exists on the control plane vnet %#q in resource group %#q", key.ResourceGroupName(cr), r.hostVirtualNetworkName, r.hostResourceGroup))
		cpPeering := r.getCPVnetPeering(*tcVnet.ID)
		_, err = r.cpAzureClientSet.VnetPeeringClient.CreateOrUpdate(ctx, r.hostResourceGroup, r.hostVirtualNetworkName, key.ResourceGroupName(cr), cpPeering)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = r.ensureVnetGatewayIsDeleted(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) ensureVnetGatewayIsDeleted(ctx context.Context, cr providerv1alpha1.AzureConfig) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if the VPN gateway still exists")

	vnetGatewaysClient, err := r.clientFactory.GetVirtualNetworkGatewaysClient(ctx, cr.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	gw, err := vnetGatewaysClient.Get(ctx, key.ResourceGroupName(cr), key.VPNGatewayName(cr))
	if IsNotFound(err) {
		// VPN gateway not found. That's our goal, all good.
		// Let's check if the public IP address still exist and delete that as well.
		r.logger.LogCtx(ctx, "level", "debug", "message", "VPN gateway does not exists")
		r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if the VPN gateway's public IP still exists")

		publicIPAddressesClient, err := r.clientFactory.GetPublicIpAddressesClient(ctx, cr.ObjectMeta)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = publicIPAddressesClient.Get(ctx, key.ResourceGroupName(cr), key.VPNGatewayPublicIPName(cr), "")
		if IsNotFound(err) {
			// That's the desired state, all good.
			r.logger.LogCtx(ctx, "level", "debug", "message", "VPN gateway's public IP does not exists")
			return nil
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "VPN gateway's public IP still exists, requesting deletion")

		_, err = publicIPAddressesClient.Delete(ctx, key.ResourceGroupName(cr), key.VPNGatewayPublicIPName(cr))
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Requested deletion of public IP %s", key.VPNGatewayPublicIPName(cr)))

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "VPN Gateway still exists")

	if gw.ProvisioningState == ProvisioningStateDeleting {
		r.logger.LogCtx(ctx, "level", "debug", "message", "VPN Gateway deletion in progress")
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if there are existing connections")

	virtualNetworkGatewayConnectionsClient, err := r.clientFactory.GetVirtualNetworkGatewayConnectionsClient(ctx, cr.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	results, err := virtualNetworkGatewayConnectionsClient.ListComplete(ctx, key.ResourceGroupName(cr))
	if err != nil {
		return microerror.Mask(err)
	}

	found := false
	for results.NotDone() {
		c := results.Value()

		if *c.VirtualNetworkGateway1.ID == *gw.ID || *c.VirtualNetworkGateway2.ID == *gw.ID {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Found VPN connection %s to be deleted", *c.Name))

			_, err := virtualNetworkGatewayConnectionsClient.Delete(ctx, key.ResourceGroupName(cr), *c.Name)
			if err != nil {
				return microerror.Mask(err)
			}

			found = true
		}

		err = results.NextWithContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if !found {
		// No connections have been found, safe to delete the VPN Gateway.
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("No connections found, deleting VPN Gateway %s", *gw.Name))

		_, err := vnetGatewaysClient.Delete(ctx, key.ResourceGroupName(cr), *gw.Name)
		if err != nil {
			return microerror.Mask(err)
		}
		return nil
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
