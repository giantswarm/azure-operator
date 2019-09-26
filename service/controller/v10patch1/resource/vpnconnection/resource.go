package vpnconnection

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v10/controllercontext"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	Name = "vpnconnectionv10"
)

// Config is the configuration required by Resource.
type Config struct {
	Logger micrologger.Logger

	Azure                    setting.Azure
	HostAzureClientSetConfig client.AzureClientSetConfig
}

// Resource manages Azure virtual network peering.
type Resource struct {
	logger micrologger.Logger

	azure                    setting.Azure
	hostAzureClientSetConfig client.AzureClientSetConfig
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.HostAzureClientSetConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostAzureClientSetConfig.%s", config, err)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	r := &Resource{
		logger: config.Logger,

		azure:                    config.Azure,
		hostAzureClientSetConfig: config.HostAzureClientSetConfig,
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
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualNetworkGatewaysClient, nil
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
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualNetworkGatewayConnectionsClient, nil
}

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

// getHostVirtualNetworkGatewaysClient return a client to interact with
// VirtualNetworkGateways on host cluster.
func (r *Resource) getHostVirtualNetworkGatewaysClient(ctx context.Context) (*network.VirtualNetworkGatewaysClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureClientSetConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.VirtualNetworkGatewaysClient, nil
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

// getHostVirtualNetworkGatewayConnectionsClient return a client to interact with
// VirtualNetworkGatewayConnections on host cluster.
func (r *Resource) getHostVirtualNetworkGatewayConnectionsClient(ctx context.Context) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureClientSetConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.VirtualNetworkGatewayConnectionsClient, nil
}
