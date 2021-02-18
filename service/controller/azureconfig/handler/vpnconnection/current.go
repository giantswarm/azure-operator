package vpnconnection

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/controller/context/resourcecanceledcontext"

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
		r.logger.Debugf(ctx, "finding host vpn gateway connection")

		resourceGroup := r.azure.HostCluster.ResourceGroup
		connectionName := key.ResourceGroupName(cr)

		connectionsClient, err := r.mcAzureClientFactory.GetVirtualNetworkGatewayConnectionsClient(ctx, key.ClusterID(&cr))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		h, err := connectionsClient.Get(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.Debugf(ctx, "did not find host vpn gateway connection")
			return connections{}, nil
		} else if err != nil {
			return connections{}, microerror.Mask(err)
		}

		if provisioningState := string(h.ProvisioningState); !key.IsFinalProvisioningState(provisioningState) {
			r.logger.Debugf(ctx, "host vpn gateway connection is in state '%s'", provisioningState)
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.Debugf(ctx, "canceling resource")

			return connections{}, nil
		}

		c.Host = h

		r.logger.Debugf(ctx, "found host vpn gateway connection")
	}

	{
		r.logger.Debugf(ctx, "finding tenant vpn gateway connection")

		resourceGroup := key.ResourceGroupName(cr)
		connectionName := r.azure.HostCluster.ResourceGroup

		connectionsClient, err := r.wcAzureClientFactory.GetVirtualNetworkGatewayConnectionsClient(ctx, key.ClusterID(&cr))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		g, err := connectionsClient.Get(ctx, resourceGroup, connectionName)
		if IsVPNGatewayConnectionNotFound(err) {
			r.logger.Debugf(ctx, "did not find tenant vpn gateway connection")
			return c, nil
		} else if err != nil {
			return c, microerror.Mask(err)
		}

		if provisioningState := string(g.ProvisioningState); !key.IsFinalProvisioningState(provisioningState) {
			r.logger.Debugf(ctx, "tenant vpn gateway connection is in state '%s'", provisioningState)
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.Debugf(ctx, "canceling resource")

			return c, nil
		}

		c.Guest = g

		r.logger.Debugf(ctx, "found tenant vpn gateway connection")
	}

	return c, nil
}
