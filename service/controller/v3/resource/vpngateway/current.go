package vpngateway

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

// GetCurrentState retrieve current vpn gateway connection from host to guest cluster.
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return connections{}, microerror.Mask(err)
	}

	var c connections
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "finding host vpn gateway connection")

		resourceGroup := r.azure.HostCluster.ResourceGroup
		connectionName := key.ResourceGroupName(customObject)

		h, err := r.getHostVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find host vpn gateway connection")
			return connections{}, nil
		} else if err != nil {
			return connections{}, microerror.Mask(err)
		}
		c.Host = *h

		r.logger.LogCtx(ctx, "level", "debug", "message", "found host vpn gateway connection")
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "finding guest vpn gateway connection")

		resourceGroup := key.ResourceGroupName(customObject)
		connectionName := r.azure.HostCluster.ResourceGroup

		g, err := r.getGuestVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find guest vpn gateway connection")
			return c, nil
		} else if err != nil {
			return c, microerror.Mask(err)
		}
		c.Guest = *g

		r.logger.LogCtx(ctx, "level", "debug", "message", "found guest vpn gateway connection")
	}

	return c, nil
}
