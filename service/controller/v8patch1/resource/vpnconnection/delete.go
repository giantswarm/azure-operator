package vpnconnection

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"

	"github.com/giantswarm/azure-operator/service/controller/v8patch1/key"
)

// NewDeletePatch provide a controller.Patch holding connections to be deleted.
func (r *Resource) NewDeletePatch(ctx context.Context, azureConfig, current, desired interface{}) (*controller.Patch, error) {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	c, err := toVPNGatewayConnections(current)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	d, err := toVPNGatewayConnections(desired)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch, err := r.newDeletePatch(ctx, a, c, d)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return patch, nil
}

// newDeletePatch use desired as delete patch since it is mostly static and more likely to be present than current.
func (r *Resource) newDeletePatch(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired connections) (*controller.Patch, error) {
	patch := controller.NewPatch()

	patch.SetDeleteChange(desired)

	return patch, nil
}

// ApplyDeleteChange perform deletion of vpn gateway connection against azure.
func (r *Resource) ApplyDeleteChange(ctx context.Context, azureConfig, change interface{}) error {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return microerror.Mask(err)
	}
	c, err := toVPNGatewayConnections(change)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.applyDeleteChange(ctx, a, c)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) applyDeleteChange(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, change connections) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring host vpn gateway connection is deleted")

	if change.isEmpty() {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured host vpn gateway connection is deleted")
		return nil
	}

	hostGatewayConnectionClient, err := r.getHostVirtualNetworkGatewayConnectionsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroup := r.azure.HostCluster.ResourceGroup
	connectionName := *change.Host.Name

	respFuture, err := hostGatewayConnectionClient.Delete(ctx, resourceGroup, connectionName)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = hostGatewayConnectionClient.DeleteResponder(respFuture.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured host vpn gateway connection is deleted")
	return nil
}
