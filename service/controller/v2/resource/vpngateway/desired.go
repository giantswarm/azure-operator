package vpngateway

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	servicecontext "github.com/giantswarm/azure-operator/service/controller/v2/context"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

// GetDesiredState return desired peering for host cluster virtual network.
// Peering resource is named after guest cluster's resource group and targeting its virtual network.
func (r *Resource) GetDesiredState(ctx context.Context, azureConfig interface{}) (interface{}, error) {
	a, err := key.ToCustomObject(azureConfig)
	if err != nil {
		return network.VirtualNetworkGatewayConnection{}, microerror.Mask(err)
	}

	vpnGatewayConnections, err := r.getDesiredState(ctx, a)
	if err != nil {
		return network.VirtualNetworkGatewayConnection{}, microerror.Mask(err)
	}

	return vpnGatewayConnections, nil
}

func (r *Resource) getDesiredState(ctx context.Context, azureConfig providerv1alpha1.AzureConfig) (connections, error) {
	sc, err := servicecontext.FromContext(ctx)
	if err != nil {
		return connections{}, microerror.Mask(err)
	}

	sharedKey := randStringBytes(128)

	return connections{
		Host: network.VirtualNetworkGatewayConnection{
			Name: to.StringPtr(key.ResourceGroupName(azureConfig)),
			VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
				ConnectionType: network.Vnet2Vnet,
				SharedKey:      to.StringPtr(sharedKey),
				VirtualNetworkGateway1: &network.VirtualNetworkGateway{
					ID: to.StringPtr(key.VPNGatewayID(r.hostAzureConfig.SubscriptionID, r.azure.HostCluster.ResourceGroup, r.azure.HostCluster.VirtualNetworkGateway)),
				},
				VirtualNetworkGateway2: &network.VirtualNetworkGateway{
					ID: to.StringPtr(key.VPNGatewayID(sc.AzureConfig.SubscriptionID, key.ResourceGroupName(azureConfig), key.VPNGatewayName(azureConfig))),
				},
			},
		},
		Guest: network.VirtualNetworkGatewayConnection{
			Name: to.StringPtr(r.azure.HostCluster.ResourceGroup),
			VirtualNetworkGatewayConnectionPropertiesFormat: &network.VirtualNetworkGatewayConnectionPropertiesFormat{
				ConnectionType: network.Vnet2Vnet,
				SharedKey:      to.StringPtr(sharedKey),
				VirtualNetworkGateway1: &network.VirtualNetworkGateway{
					ID: to.StringPtr(key.VPNGatewayID(sc.AzureConfig.SubscriptionID, key.ResourceGroupName(azureConfig), key.VPNGatewayName(azureConfig))),
				},
				VirtualNetworkGateway2: &network.VirtualNetworkGateway{
					ID: to.StringPtr(key.VPNGatewayID(r.hostAzureConfig.SubscriptionID, r.azure.HostCluster.ResourceGroup, r.azure.HostCluster.VirtualNetworkGateway)),
				},
			},
		},
	}, nil
}
