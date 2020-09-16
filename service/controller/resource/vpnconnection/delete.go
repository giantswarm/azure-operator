package vpnconnection

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/resource/crud"
)

// NewDeletePatch provide a crud.Patch holding connections to be deleted.
func (r *Resource) NewDeletePatch(ctx context.Context, azureConfig, current, desired interface{}) (*crud.Patch, error) {
	d, err := toVPNGatewayConnections(desired)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := r.newDeletePatch(d)

	return patch, nil
}

// newDeletePatch use desired as delete patch since it is mostly static and more likely to be present than current.
func (r *Resource) newDeletePatch(desired connections) *crud.Patch {
	patch := crud.NewPatch()

	patch.SetDeleteChange(desired)

	return patch
}

// ApplyDeleteChange perform deletion of vpn gateway connection against azure.
func (r *Resource) ApplyDeleteChange(ctx context.Context, azureConfig, change interface{}) error {
	c, err := toVPNGatewayConnections(change)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.applyDeleteChange(ctx, c)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) applyDeleteChange(ctx context.Context, change connections) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring host vpn gateway connection is deleted")

	if change.isEmpty() {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured host vpn gateway connection is deleted")
		return nil
	}

	resourceGroup := r.azure.HostCluster.ResourceGroup
	connectionName := *change.Host.Name

	respFuture, err := r.cpVirtualNetworkGatewayConnectionsClient.Delete(ctx, resourceGroup, connectionName)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = r.cpVirtualNetworkGatewayConnectionsClient.DeleteResponder(respFuture.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured host vpn gateway connection is deleted")
	return nil
}
