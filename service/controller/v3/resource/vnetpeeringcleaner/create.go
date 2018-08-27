package vnetpeeringcleaner

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

// EnsureCreated ensure that vnetpeering resource are deleted,
// since they are no longer in use in this version.
func (r Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deletion of host vnetpeering")

	azureConfig, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	vnetPeeringHostClient, err := r.getVnetPeeringHostClient()
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroupName := r.azure.HostCluster.ResourceGroup
	vnetName := r.azure.HostCluster.ResourceGroup
	peeringName := key.ResourceGroupName(azureConfig)
	err = r.deletePeering(ctx, vnetPeeringHostClient, resourceGroupName, vnetName, peeringName)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deletion of host vnetpeering")

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deletion of guest vnetpeering")

	vnetPeeringGuestClient, err := r.getVnetPeeringGuestClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroupName = key.ResourceGroupName(azureConfig)
	vnetName = key.VnetName(azureConfig)
	peeringName = r.azure.HostCluster.ResourceGroup
	err = r.deletePeering(ctx, vnetPeeringGuestClient, resourceGroupName, vnetName, peeringName)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deletion of guest vnetpeering")

	return nil
}

func (r Resource) deletePeering(ctx context.Context, vnetPeeringClient *network.VirtualNetworkPeeringsClient, resourceGroupName, vnetName, peeringName string) error {
	respFuture, err := vnetPeeringClient.Delete(ctx, resourceGroupName, vnetName, peeringName)
	r.logger.LogCtx(ctx, "level", "debug", "message", "deleting", "error", fmt.Sprintf("%#v", err))
	if IsNotFound(err) {
		// fall through
		r.logger.LogCtx(ctx, "level", "debug", "message", "IsNotFound", "error", fmt.Sprintf("%#v", err))
	} else if err != nil {
		return microerror.Mask(err)
	}

	// DeleteResponder ensure that response body is closed.
	res, err := vnetPeeringClient.DeleteResponder(respFuture.Response())
	if client.ResponseWasNotFound(res) {
		// fall through
		r.logger.LogCtx(ctx, "level", "debug", "message", "ResponseWasNotFound", "error", fmt.Sprintf("%#v", err))
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
