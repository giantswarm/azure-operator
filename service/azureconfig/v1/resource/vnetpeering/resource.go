package vnetpeering

import (
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
)

const (
	Name = "vnetpeeringv1"
)

// Config is the configuration required by Resource.
type Config struct {
	AzureConfig client.AzureConfig
	Logger      micrologger.Logger
}

// Resource manages Azure virtual network peering.
type Resource struct {
	azureConfig client.AzureConfig
	logger      micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig.%s", err)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	r := &Resource{
		azureConfig: config.AzureConfig,
		logger:      config.Logger,
	}

	return r, nil
}

func (r Resource) Name() string {
	return Name
}

// getVnetPeeringClient return an azure client to interact with VirtualNetworkPeering resource.
func (r Resource) getVnetPeeringClient() (*network.VirtualNetworkPeeringsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "creating Azure virtual network peering client")
	}

	return azureClients.VnetPeeringClient, nil
}
