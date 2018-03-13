package vnetpeering

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	Name = "vnetpeeringv1"
)

// Config is the configuration required by Resource
type Config struct {
	AzureConfig client.AzureConfig
	Logger      micrologger.Logger
}

// Resource manages Azure virtual network peering
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

// getVnetPeeringClient return an azure client to interact with VirtualNetworkPeering resource
func (r Resource) getVnetPeeringClient() (*network.VirtualNetworkPeeringsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "creating Azure virtual network peering client")
	}

	return azureClients.VnetPeeringClient, nil
}

// toVnePeering convert v to network.VirtualNetworkPeering
// If v is nil and empty network.VirtualNetworkPeering is returned
func toVnetPeering(v interface{}) (network.VirtualNetworkPeering, error) {
	if v == nil {
		return network.VirtualNetworkPeering{}, nil
	}

	vnetPeering, ok := v.(network.VirtualNetworkPeering)
	if !ok {
		return network.VirtualNetworkPeering{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", network.VirtualNetworkPeering{}, v)
	}

	return vnetPeering, nil
}
