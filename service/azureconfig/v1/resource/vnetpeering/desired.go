package vnetpeering

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/key"
	"github.com/giantswarm/microerror"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

// GetDesiredState return desired peering for host cluster virtual network.
// Peering resource is named after cluster's resource group and targeting its virtual network.
func (r Resource) GetDesiredState(ctx context.Context, azureConfig interface{}) (interface{}, error) {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return network.VirtualNetworkPeering{}, microerror.Maskf(err, "GetDesiredState")
	}

	vnetPeering, err := r.getDesiredState(ctx, a)
	if err != nil {
		return network.VirtualNetworkPeering{}, microerror.Maskf(err, "GetDesiredState")
	}

	return vnetPeering, nil
}

func (r *Resource) getDesiredState(ctx context.Context, azureConfig providerv1alpha1.AzureConfig) (network.VirtualNetworkPeering, error) {
	return network.VirtualNetworkPeering{
		Name: to.StringPtr(key.ResourceGroupName(azureConfig)),
		VirtualNetworkPeeringPropertiesFormat: &network.VirtualNetworkPeeringPropertiesFormat{
			AllowVirtualNetworkAccess: to.BoolPtr(true),
			RemoteVirtualNetwork: &network.SubResource{
				ID: to.StringPtr(key.VNetID(azureConfig, r.azureConfig.SubscriptionID)),
			},
		},
	}, nil
}
