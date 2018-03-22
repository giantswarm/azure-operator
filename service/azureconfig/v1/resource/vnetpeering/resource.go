package vnetpeering

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/azureconfig/config"
)

const (
	Name = "vnetpeeringv1"
)

// Config is the configuration required by Resource.
type Config struct {
	Logger micrologger.Logger

	Azure       config.Azure
	AzureConfig client.AzureConfig
}

// Resource manages Azure virtual network peering.
type Resource struct {
	logger micrologger.Logger

	azure       config.Azure
	azureConfig client.AzureConfig
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureConfig.%s", config, err)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	r := &Resource{
		logger: config.Logger,

		azure:       config.Azure,
		azureConfig: config.AzureConfig,
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
