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
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return connections{}, microerror.Mask(err)
	}

	var hostVPNGatewayConnection *network.VirtualNetworkGatewayConnection
	{
		resourceGroup := r.azure.HostCluster.ResourceGroup
		connectionName := key.ResourceGroupName(customObject)

		hostVPNGatewayConnection, err = r.getHostVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "host vpn gateway connection not found")
			return connections{}, nil
		} else if err != nil {
			return connections{}, microerror.Mask(err)
		}

		if provisioningState := *hostVPNGatewayConnection.ProvisioningState; !key.IsFinalProvisioningState(provisioningState) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("host vpn gateway connection is in state '%s'", provisioningState))
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")
			return connections{}, nil
		}
	}
	result := connections{
		Host: *hostVPNGatewayConnection,
	}

	var guestVPNGatewayConnection *network.VirtualNetworkGatewayConnection
	{
		resourceGroup := key.ResourceGroupName(customObject)
		connectionName := r.azure.HostCluster.ResourceGroup

		guestVPNGatewayConnection, err = r.getGuestVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "guest vpn gateway connection not found")
			return result, nil
		} else if err != nil {
			return result, microerror.Mask(err)
		}

		if provisioningState := *guestVPNGatewayConnection.ProvisioningState; !key.IsFinalProvisioningState(provisioningState) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("guest vpn gateway connection is in state '%s'", provisioningState))
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")
			return result, nil
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "vpn gateway connections found")
	return connections{
		Host:  *hostVPNGatewayConnection,
		Guest: *guestVPNGatewayConnection,
	}, nil
}
