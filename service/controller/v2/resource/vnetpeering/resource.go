package vnetpeering

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	servicecontext "github.com/giantswarm/azure-operator/service/controller/v2/context"
)

const (
	Name = "vnetpeeringv2"
)

// Config is the configuration required by Resource.
type Config struct {
	Logger micrologger.Logger

	Azure           setting.Azure
	HostAzureConfig client.AzureClientSetConfig
}

// Resource manages Azure virtual network peering.
type Resource struct {
	logger micrologger.Logger

	azure           setting.Azure
	hostAzureConfig client.AzureClientSetConfig
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.HostAzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostAzureConfig.%s", config, err)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	r := &Resource{
		logger: config.Logger,

		azure:           config.Azure,
		hostAzureConfig: config.HostAzureConfig,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

// getVirtualNetworksClient return an azure client to interact with
// VirtualNetworks resources.
func (r *Resource) getVirtualNetworksClient(ctx context.Context) (*network.VirtualNetworksClient, error) {
	sc, err := servicecontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualNetworkClient, nil
}

// getVnetPeeringClient return an azure client to interact with
// VirtualNetworkPeering resources.
func (r *Resource) getVnetPeeringClient(ctx context.Context) (*network.VirtualNetworkPeeringsClient, error) {
	sc, err := servicecontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VnetPeeringClient, nil
}
