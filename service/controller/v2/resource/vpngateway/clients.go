package vpngateway

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/microerror"

	servicecontext "github.com/giantswarm/azure-operator/service/controller/v2/context"
)

func (r *Resource) getVirtualNetworkGateway(ctx context.Context, resourceGroup, vpnGatewayName string) (*network.VirtualNetworkGateway, error) {
	gatewayClient, err := r.getVirtualNetworkGatewaysClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	gateway, err := gatewayClient.Get(ctx, resourceGroup, vpnGatewayName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &gateway, nil
}

func (r *Resource) getHostVirtualNetworkGatewayConnection(ctx context.Context, resourceGroup, vpnGatewayConnectionName string) (*network.VirtualNetworkGatewayConnection, error) {
	gatewayConnectionClient, err := r.getHostVirtualNetworkGatewayConnectionsClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	connection, err := gatewayConnectionClient.Get(ctx, resourceGroup, vpnGatewayConnectionName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &connection, nil
}

func (r *Resource) getGuestVirtualNetworkGatewayConnection(ctx context.Context, resourceGroup, vpnGatewayConnectionName string) (*network.VirtualNetworkGatewayConnection, error) {
	gatewayConnectionClient, err := r.getGuestVirtualNetworkGatewayConnectionsClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	connection, err := gatewayConnectionClient.Get(ctx, resourceGroup, vpnGatewayConnectionName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &connection, nil
}

// getVirtualNetworkGatewaysClient return an azure client to interact with
// VirtualNetworkGateways resources.
func (r *Resource) getVirtualNetworkGatewaysClient(ctx context.Context) (*network.VirtualNetworkGatewaysClient, error) {
	sc, err := servicecontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualNetworkGatewaysClient, nil
}

// getVirtualNetworkGatewayConnectionsClient return an azure client to interact with
// VirtualNetworkGateways connections resources.
func (r *Resource) getHostVirtualNetworkGatewayConnectionsClient(ctx context.Context) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.VirtualNetworkGatewayConnectionsClient, nil
}

// getVirtualNetworkGatewayConnectionsClient return an azure client to interact with
// VirtualNetworkGateways connections resources.
func (r *Resource) getGuestVirtualNetworkGatewayConnectionsClient(ctx context.Context) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	sc, err := servicecontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualNetworkGatewayConnectionsClient, nil
}
