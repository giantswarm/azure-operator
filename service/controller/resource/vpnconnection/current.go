package vpnconnection

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// GetCurrentState retrieve current vpn gateway connection from host to tenant
// cluster.
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return connections{}, microerror.Mask(err)
	}

	var c connections
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "finding host vpn gateway connection")

		resourceGroup := r.azure.HostCluster.ResourceGroup
		connectionName := key.ResourceGroupName(cr)

		h, err := r.getHostVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find host vpn gateway connection")
			return connections{}, nil
		} else if err != nil {
			return connections{}, microerror.Mask(err)
		}

		if provisioningState := string(h.ProvisioningState); !key.IsFinalProvisioningState(provisioningState) {
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

		resourceGroup := key.ResourceGroupName(cr)
		connectionName := r.azure.HostCluster.ResourceGroup

		g, err := r.getGuestVirtualNetworkGatewayConnection(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find tenant vpn gateway connection")
			return c, nil
		} else if err != nil {
			return c, microerror.Mask(err)
		}

		if provisioningState := string(g.ProvisioningState); !key.IsFinalProvisioningState(provisioningState) {
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
