package vpngateway

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/v4/key"
)

// GetDesiredState return desired vpn gateway connections.
func (r *Resource) GetDesiredState(ctx context.Context, azureConfig interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return connections{}, microerror.Mask(err)
	}

	var (
		hostVPNGateway  *network.VirtualNetworkGateway
		guestVPNGateway *network.VirtualNetworkGateway
	)
	// Do not check for vpn gateway when deleting.
	// As we do not require guest cluster vpn gateway to be ready in order to
	// delete connection from host cluster vpn gateway.
	if !key.IsDeleted(customObject) {
		// In order to make vpn gateway connection work we need 2 vpn gateway.
		// One on the host cluster and one on the guest cluster.
		// Here we check for vpn gateways readiness.
		// In case one of the vpn gateway is not ready we cancel the resource
		// and try again on the next resync period.
		{

			resourceGroup := key.ResourceGroupName(customObject)
			vpnGatewayName := key.VPNGatewayName(customObject)

			guestVPNGateway, err = r.getGuestVirtualNetworkGateway(ctx, resourceGroup, vpnGatewayName)
			if IsVPNGatewayNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "guest vpn gateway was not found")
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

				return connections{}, nil
			} else if err != nil {
				return connections{}, microerror.Mask(err)
			}

			if provisioningState := *guestVPNGateway.ProvisioningState; provisioningState != "Succeeded" {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("guest vpn gateway is in state '%s'", provisioningState))
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

				return connections{}, nil
			}
		}

		{
			resourceGroup := r.azure.HostCluster.ResourceGroup
			vpnGatewayName := r.azure.HostCluster.VirtualNetworkGateway

			hostVPNGateway, err = r.getHostVirtualNetworkGateway(ctx, resourceGroup, vpnGatewayName)
			if IsVPNGatewayNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "host vpn gateway was not found")
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

				return connections{}, nil
			} else if err != nil {
				return connections{}, microerror.Mask(err)
			}

			if provisioningState := *hostVPNGateway.ProvisioningState; provisioningState != "Succeeded" {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("host vpn gateway is in state '%s'", provisioningState))
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

				return connections{}, nil
			}
		}
	}

	vpnGatewayConnections, err := r.getDesiredState(ctx, customObject, guestVPNGateway, hostVPNGateway)
	if err != nil {
		return connections{}, microerror.Mask(err)
	}

	return vpnGatewayConnections, nil
}

func (r *Resource) getDesiredState(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, guestVPNGateway, hostVPNGateway *network.VirtualNetworkGateway) (connections, error) {
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
	}, nil
}
