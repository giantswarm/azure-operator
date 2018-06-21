package vpngateway

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

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

	var hostVPNGatewayConnection *network.VirtualNetworkGatewayConnection
	{
		resourceGroup := r.azure.HostCluster.ResourceGroup
		connectionName := key.ResourceGroupName(customObject)

		hostVPNGatewayConnection, err = r.getHostVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "host vpn gateway connection not found")
			err = nil
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
	result = connections{
		Host: *hostVPNGatewayConnection,
	}

	var guestVPNGatewayConnection *network.VirtualNetworkGatewayConnection
	{
		resourceGroup := key.ResourceGroupName(customObject)
		connectionName := r.azure.HostCluster.ResourceGroup

		guestVPNGatewayConnection, err = r.getGuestVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "guest vpn gateway connection not found")
			err = nil
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
	result = connections{
		Host:  *hostVPNGatewayConnection,
		Guest: *guestVPNGatewayConnection,
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "vpn gateway connections found")
	return
}
