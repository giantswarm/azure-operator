package vnetpeering

import (
	"context"
	// "fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/framework"
)

func (r Resource) NewDeletePatch(ctx context.Context, azureConfig, current, desired interface{}) (*framework.Patch, error) {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}
	c, err := toVnetPeering(current)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}
	d, err := toVnetPeering(desired)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}

	return r.newDeletePatch(ctx, a, c, d)
}

func (r Resource) newDeletePatch(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, current, desired network.VirtualNetworkPeering) (*framework.Patch, error) {
	patch := framework.NewPatch()
	patch.SetDeleteChange(current)
	return patch, nil
}

// ApplyDeleteChange perform the host cluster virtual network peering deletion against azure.
func (r Resource) ApplyDeleteChange(ctx context.Context, azureConfig, change interface{}) error {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return microerror.Maskf(err, "ApplyDeleteChange")
	}
	c, err := toVnetPeering(change)
	if err != nil {
		return microerror.Maskf(err, "ApplyDeleteChange")
	}

	return r.applyDeleteChange(ctx, a, c)
}

func (r Resource) applyDeleteChange(ctx context.Context, azureConfig providerv1alpha1.AzureConfig, change network.VirtualNetworkPeering) error {
	r.logger.LogCtx(ctx, "debug", "deleting host vnet peering")

	vnetPeeringClient, err := r.getVnetPeeringClient()
	if err != nil {
		return microerror.Maskf(err, "deleting host vnet peering")
	}

	_, err = vnetPeeringClient.Delete(ctx, key.HostClusterResourceGroup(azureConfig), key.HostClusterResourceGroup(azureConfig), key.ResourceGroupName(azureConfig))
	if err != nil {
		return microerror.Maskf(err, "deleting host vnet peering")
	}

	r.logger.LogCtx(ctx, "debug", "deleting host vnet peering: deleted")
	return nil
}
