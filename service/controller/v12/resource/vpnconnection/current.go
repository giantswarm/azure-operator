package vpnconnection

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/v11/key"
)

// GetCurrentState retrieve current vpn gateway connection from host to tenant
// cluster.
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

		if provisioningState := *h.ProvisioningState; !key.IsFinalProvisioningState(provisioningState) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("host vpn gateway connection is in state '%s'", provisioningState))
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return connections{}, nil
		}

		c.Host = *h

		r.logger.LogCtx(ctx, "level", "debug", "message", "found host vpn gateway connection")
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "finding tenant vpn gateway connection")

		resourceGroup := key.ResourceGroupName(customObject)
		connectionName := r.azure.HostCluster.ResourceGroup

		g, err := r.getGuestVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find tenant vpn gateway connection")
			return c, nil
		} else if err != nil {
			return c, microerror.Mask(err)
		}

		if provisioningState := *g.ProvisioningState; !key.IsFinalProvisioningState(provisioningState) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tenant vpn gateway connection is in state '%s'", provisioningState))
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return c, nil
		}

		c.Guest = *g

		r.logger.LogCtx(ctx, "level", "debug", "message", "found tenant vpn gateway connection")
	}

	return c, nil
}
