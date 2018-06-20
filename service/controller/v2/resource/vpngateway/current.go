package vpngateway

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

// GetCurrentState retrieve current vpn gateway connection from host to guest cluster.
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (result interface{}, err error) {
	result = connections{}

	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		err = microerror.Mask(err)
		return
	}

	// Do not check for vpn gateway when deleting.
	// As we do not require guest cluster vpn gateway to be ready in order to
	// delete connection from host cluster vpn gateway.
	if !key.IsDeleted(customObject) {
		var vpnGateway *network.VirtualNetworkGateway
		{
			// In order to make vpn gateway connection work we need 2 vpn gateway.
			// We assume the host cluster vpn gateway is ready.
			// Here we check for guest cluster vpn gateway readiness.
			// In case vpn gateway is not ready we cancel the resource
			// and try again on the next resync period.

			guestResourceGroup := key.ResourceGroupName(customObject)
			vpnGatewayName := key.VPNGatewayName(customObject)

			vpnGateway, err = r.getVirtualNetworkGateway(ctx, guestResourceGroup, vpnGatewayName)
			if client.ResponseWasNotFound(vpnGateway.Response) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "vpn gateway was not found")
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

				return
			} else if err != nil {
				err = microerror.Mask(err)
				return
			}

			if provisioningState := *vpnGateway.ProvisioningState; provisioningState != "Succeeded" {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("vpn gateway is in state '%s'", provisioningState))
				resourcecanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

				return
			}
		}

	}

	var hostVPNGatewayConnection *network.VirtualNetworkGatewayConnection
	{
		resourceGroup := r.azure.HostCluster.ResourceGroup
		connectionName := key.ResourceGroupName(customObject)

		hostVPNGatewayConnection, err = r.getHostVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if client.ResponseWasNotFound(hostVPNGatewayConnection.Response) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "host vpn gateway connection not found")

			return
		} else if err != nil {
			err = microerror.Mask(err)
			return
		}

		if provisioningState := *hostVPNGatewayConnection.ProvisioningState; !key.IsFinalProvisioningState(provisioningState) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("host vpn gateway connection is in state '%s'", provisioningState))
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return
		}
	}

	var guestVPNGatewayConnection *network.VirtualNetworkGatewayConnection
	{
		resourceGroup := key.ResourceGroupName(customObject)
		connectionName := r.azure.HostCluster.ResourceGroup

		guestVPNGatewayConnection, err = r.getGuestVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if client.ResponseWasNotFound(guestVPNGatewayConnection.Response) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "guest vpn gateway connection not found")

			return
		} else if err != nil {
			err = microerror.Mask(err)
			return
		}

		if provisioningState := *guestVPNGatewayConnection.ProvisioningState; !key.IsFinalProvisioningState(provisioningState) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("guest vpn gateway connection is in state '%s'", provisioningState))
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "vpn gateway connections found")
	result = connections{
		Host:  *hostVPNGatewayConnection,
		Guest: *guestVPNGatewayConnection,
	}

	return
}
