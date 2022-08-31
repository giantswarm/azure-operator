package vnetpeering

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/finalizerskeptcontext"

	"github.com/giantswarm/azure-operator/v6/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if the vnet peering exists on the control plane vnet")

		_, err = r.cpAzureClientSet.VnetPeeringClient.Get(ctx, r.mcVirtualNetworkName, r.mcVirtualNetworkName, key.ResourceGroupName(cr))
		if IsNotFound(err) {
			// This is what we want, all good.
			r.logger.LogCtx(ctx, "level", "debug", "message", "Vnet peering doesn't exist on the control plane vnet")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "Vnet peering still exists on the control plane vnet")
	}

	// Keep the finalizer until as long as the peering connection still exists.
	finalizerskeptcontext.SetKept(ctx)

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "Requesting deletion vnet peering on the control plane vnet")

		_, err = r.cpAzureClientSet.VnetPeeringClient.Delete(ctx, r.mcVirtualNetworkName, r.mcVirtualNetworkName, key.ResourceGroupName(cr))
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "Requested deletion of vnet peering on the control plane vnet")
	}

	return nil
}
