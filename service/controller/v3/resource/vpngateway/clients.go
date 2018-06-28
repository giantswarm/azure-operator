package vpngateway

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v3/controllercontext"
)

func (r *Resource) getHostVirtualNetworkGateway(ctx context.Context, resourceGroup, vpnGatewayName string) (*network.VirtualNetworkGateway, error) {
	gatewayClient, err := r.getHostVirtualNetworkGatewaysClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	gateway, err := gatewayClient.Get(ctx, resourceGroup, vpnGatewayName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &gateway, nil
}

func (r *Resource) getGuestVirtualNetworkGateway(ctx context.Context, resourceGroup, vpnGatewayName string) (*network.VirtualNetworkGateway, error) {
	gatewayClient, err := r.getGuestVirtualNetworkGatewaysClient(ctx)
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

// getHostVirtualNetworkGatewaysClient return a client to interact with
// VirtualNetworkGateways on host cluster.
func (r *Resource) getHostVirtualNetworkGatewaysClient(ctx context.Context) (*network.VirtualNetworkGatewaysClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.VirtualNetworkGatewaysClient, nil
}

// getGuestVirtualNetworkGatewaysClient return a client to interact with
// VirtualNetworkGateways on guest cluster.
func (r *Resource) getGuestVirtualNetworkGatewaysClient(ctx context.Context) (*network.VirtualNetworkGatewaysClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualNetworkGatewaysClient, nil
}

// getHostVirtualNetworkGatewayConnectionsClient return a client to interact with
// VirtualNetworkGatewayConnections on host cluster.
func (r *Resource) getHostVirtualNetworkGatewayConnectionsClient(ctx context.Context) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.VirtualNetworkGatewayConnectionsClient, nil
}

// getGuestVirtualNetworkGatewayConnectionsClient return a client to interact with
// VirtualNetworkGatewayConnections on guest cluster.
func (r *Resource) getGuestVirtualNetworkGatewayConnectionsClient(ctx context.Context) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualNetworkGatewayConnectionsClient, nil
}
