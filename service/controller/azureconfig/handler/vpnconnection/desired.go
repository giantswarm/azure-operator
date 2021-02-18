package vpnconnection

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// GetDesiredState return desired vpn gateway connections.
func (r *Resource) GetDesiredState(ctx context.Context, azureConfig interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(azureConfig)
	if err != nil {
		return connections{}, microerror.Mask(err)
	}

	var (
		hostVPNGateway  *network.VirtualNetworkGateway
		guestVPNGateway *network.VirtualNetworkGateway
	)
	// Do not check for vpn gateway when deleting. As we do not require tenant
	// cluster vpn gateway to be ready in order to delete connection from host
	// cluster vpn gateway.
	if !key.IsDeleted(&cr) {
		// In order to make vpn gateway connection work we need 2 vpn gateway. One
		// on the host cluster and one on the tenant cluster. Here we check for vpn
		// gateways readiness. In case one of the vpn gateway is not ready we cancel
		// the resource and try again on the next resync period.
		{

			resourceGroup := key.ResourceGroupName(cr)
			vpnGatewayName := key.VPNGatewayName(cr)

			gatewayClient, err := r.mcAzureClientFactory.GetVirtualNetworkGatewaysClient(ctx, key.ClusterID(&cr))
			if err != nil {
				return nil, microerror.Mask(err)
			}

			guestVPNGateway, err := gatewayClient.Get(ctx, resourceGroup, vpnGatewayName)
			if IsVPNGatewayNotFound(err) {
				r.logger.Debugf(ctx, "tenant vpn gateway was not found")
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.Debugf(ctx, "canceling resource")

				return connections{}, nil
			} else if err != nil {
				return connections{}, microerror.Mask(err)
			}

			provisioningState := guestVPNGateway.ProvisioningState
			if provisioningState != "Succeeded" {
				r.logger.Debugf(ctx, "tenant vpn gateway is in state '%s'", provisioningState)
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.Debugf(ctx, "canceling resource")

				return connections{}, nil
			}
		}

		{
			resourceGroup := r.azure.HostCluster.ResourceGroup
			vpnGatewayName := r.azure.HostCluster.VirtualNetworkGateway

			gatewayClient, err := r.wcAzureClientFactory.GetVirtualNetworkGatewaysClient(ctx, key.ClusterID(&cr))
			if err != nil {
				return nil, microerror.Mask(err)
			}

			hostVPNGateway, err := gatewayClient.Get(ctx, resourceGroup, vpnGatewayName)
			if IsVPNGatewayNotFound(err) {
				r.logger.Debugf(ctx, "host vpn gateway was not found")
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.Debugf(ctx, "canceling resource")

				return connections{}, nil
			} else if err != nil {
				return connections{}, microerror.Mask(err)
			}

			if provisioningState := string(hostVPNGateway.ProvisioningState); provisioningState != "Succeeded" {
				r.logger.Debugf(ctx, "host vpn gateway is in state '%s'", provisioningState)
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.Debugf(ctx, "canceling resource")

				return connections{}, nil
			}
		}
	}

	return r.getDesiredState(cr, guestVPNGateway, hostVPNGateway), nil
}

func (r *Resource) getDesiredState(azureConfig providerv1alpha1.AzureConfig, guestVPNGateway, hostVPNGateway *network.VirtualNetworkGateway) connections {
	sharedKey := randStringBytes(128)

	host := network.VirtualNetworkGatewayConnection{
		Name: to.StringPtr(key.ResourceGroupName(azureConfig)),
		VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
			ConnectionType:         network.Vnet2Vnet,
			SharedKey:              to.StringPtr(sharedKey),
			VirtualNetworkGateway1: hostVPNGateway,
			VirtualNetworkGateway2: guestVPNGateway,
		},
	}

	guest := network.VirtualNetworkGatewayConnection{
		Name: to.StringPtr(r.azure.HostCluster.ResourceGroup),
		VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
			ConnectionType:         network.Vnet2Vnet,
			SharedKey:              to.StringPtr(sharedKey),
			VirtualNetworkGateway1: guestVPNGateway,
			VirtualNetworkGateway2: hostVPNGateway,
		},
	}

	if hostVPNGateway != nil {
		host.Location = hostVPNGateway.Location
	}

	if guestVPNGateway != nil {
		guest.Location = guestVPNGateway.Location
	}

	return connections{
		Host:  host,
		Guest: guest,
	}
}
