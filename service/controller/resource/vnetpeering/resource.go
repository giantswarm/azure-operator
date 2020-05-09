package vnetpeering

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v3/client"
	"github.com/giantswarm/azure-operator/v3/service/controller/controllercontext"
)

const (
	Name = "vnetpeering"
)

type Config struct {
	HostAzureClientSetConfig client.AzureClientSetConfig
	HostResourceGroup        string
	HostVirtualNetworkName   string
	Logger                   micrologger.Logger
}

type Resource struct {
	hostAzureClientSetConfig client.AzureClientSetConfig
	hostResourceGroup        string
	hostVirtualNetworkName   string
	logger                   micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if err := config.HostAzureClientSetConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.HostAzureClientSetConfig.%s", err)
	}

	if config.HostResourceGroup == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostResourceGroup must not be empty", config)
	}

	if config.HostVirtualNetworkName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostVirtualNetworkName must not be empty", config)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		hostAzureClientSetConfig: config.HostAzureClientSetConfig,
		hostResourceGroup:        config.HostResourceGroup,
		hostVirtualNetworkName:   config.HostVirtualNetworkName,
		logger:                   config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getCPVnetPeeringsClient() (*network.VirtualNetworkPeeringsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureClientSetConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.VnetPeeringClient, nil
}

func (r *Resource) getCPVnetClient() (*network.VirtualNetworksClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureClientSetConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.VirtualNetworkClient, nil
}

func (r *Resource) getPublicIPAddressesClient(ctx context.Context) (*network.PublicIPAddressesClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.PublicIPAddressesClient, nil
}

func (r *Resource) getTCVnetPeeringsClient(ctx context.Context) (*network.VirtualNetworkPeeringsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VnetPeeringClient, nil
}

func (r *Resource) getTCVnetClient(ctx context.Context) (*network.VirtualNetworksClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkClient, nil
}

func (r *Resource) getVnetGatewaysClient(ctx context.Context) (*network.VirtualNetworkGatewaysClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkGatewaysClient, nil
}

func (r *Resource) getVnetGatewaysConnectionsClient(ctx context.Context) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkGatewayConnectionsClient, nil
}
