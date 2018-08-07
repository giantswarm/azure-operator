package vnetpeeringcleaner

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

// EnsureCreated ensure that vnetpeering resource are deleted,
// since they are no longer in use in this version.
func (r Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring delete host vnetpeering")

	azureConfig, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	vnetPeeringClient, err := r.getVnetPeeringClient()
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroupName := r.azure.HostCluster.ResourceGroup
	vnetName := r.azure.HostCluster.ResourceGroup
	peeringName := key.ResourceGroupName(azureConfig)
	respFuture, err := vnetPeeringClient.Delete(ctx, resourceGroupName, vnetName, peeringName)
	if err != nil {
		return microerror.Mask(err)
	}

	// DeleteResponder ensure that response body is closed.
	_, err = vnetPeeringClient.DeleteResponder(respFuture.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured delete host vnetpeering")

	return nil
}
