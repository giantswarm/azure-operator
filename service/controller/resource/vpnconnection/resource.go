package vpnconnection

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"
)

const (
	Name = "vpnconnection"
)

// Config is the configuration required by Resource.
type Config struct {
	Azure                                    setting.Azure
	Logger                                   micrologger.Logger
	CPVirtualNetworkGatewaysClient           network.VirtualNetworkGatewaysClient
	CPVirtualNetworkGatewayConnectionsClient network.VirtualNetworkGatewayConnectionsClient
}

// Resource manages Azure virtual network peering.
type Resource struct {
	azure                                    setting.Azure
	logger                                   micrologger.Logger
	cpVirtualNetworkGatewaysClient           network.VirtualNetworkGatewaysClient
	cpVirtualNetworkGatewayConnectionsClient network.VirtualNetworkGatewayConnectionsClient
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	r := &Resource{
		azure:                                    config.Azure,
		cpVirtualNetworkGatewaysClient:           config.CPVirtualNetworkGatewaysClient,
		cpVirtualNetworkGatewayConnectionsClient: config.CPVirtualNetworkGatewayConnectionsClient,
		logger:                                   config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
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

// getGuestVirtualNetworkGatewaysClient return a client to interact with
// VirtualNetworkGateways on guest cluster.
func (r *Resource) getGuestVirtualNetworkGatewaysClient(ctx context.Context) (*network.VirtualNetworkGatewaysClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkGatewaysClient, nil
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

// getGuestVirtualNetworkGatewayConnectionsClient return a client to interact with
// VirtualNetworkGatewayConnections on guest cluster.
func (r *Resource) getGuestVirtualNetworkGatewayConnectionsClient(ctx context.Context) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkGatewayConnectionsClient, nil
}

func (r *Resource) getHostVirtualNetworkGateway(ctx context.Context, resourceGroup, vpnGatewayName string) (*network.VirtualNetworkGateway, error) {
	gateway, err := r.cpVirtualNetworkGatewaysClient.Get(ctx, resourceGroup, vpnGatewayName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &gateway, nil
}

func (r *Resource) getHostVirtualNetworkGatewayConnection(ctx context.Context, resourceGroup, vpnGatewayConnectionName string) (*network.VirtualNetworkGatewayConnection, error) {
	connection, err := r.cpVirtualNetworkGatewayConnectionsClient.Get(ctx, resourceGroup, vpnGatewayConnectionName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &connection, nil
}
