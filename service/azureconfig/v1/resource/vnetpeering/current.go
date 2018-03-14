package vnetpeering

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/key"
	"github.com/giantswarm/microerror"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
)

// GetCurrentState retrieve the current host cluster virtual network peering resource from azure.
func (r Resource) GetCurrentState(ctx context.Context, azureConfig interface{}) (interface{}, error) {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "GetCurrentState")
	}

	return r.getCurrentState(ctx, a)
}

func (r Resource) getCurrentState(ctx context.Context, azureConfig providerv1alpha1.AzureConfig) (interface{}, error) {
	vnetPeeringClient, err := r.getVnetPeeringClient()
	if err != nil {
		return nil, microerror.Maskf(err, "getCurrentState")
	}

	vnetPeering, err := vnetPeeringClient.Get(ctx, key.HostClusterResourceGroup(azureConfig), key.HostClusterResourceGroup(azureConfig), key.ResourceGroupName(azureConfig))
	if client.ResponseWasNotFound(vnetPeering.Response) {
		return network.VirtualNetworkPeering{}, nil
	}

	return vnetPeering, err
}
