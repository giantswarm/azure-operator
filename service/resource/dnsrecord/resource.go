package dnsrecord

import (
	"github.com/Azure/azure-sdk-for-go/arm/dns"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/azure-operator/client"
)

const (
	// Name is the identifier of the resource.
	Name = "dnsrecord"
)

// Config is the resource group Resource configuration.
type Config struct {
	// Dependencies.

	AzureConfig client.AzureConfig
	Logger      micrologger.Logger
}

// DefaultConfig provides a default configuration to create a new resource by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		AzureConfig: client.DefaultAzureConfig(),
		Logger:      nil,
	}
}

// Resource manages Azure resource groups.
type Resource struct {
	// Dependencies.

	azureConfig client.AzureConfig
	logger      micrologger.Logger
}

// New creates a new configured resource group resource.
func New(config Config) (*Resource, error) {
	// Dependencies.
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig.%s", err)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	newService := &Resource{
		azureConfig: config.AzureConfig,
		logger: config.Logger.With(
			"resource", Name,
		),
	}

	return newService, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// Underlying returns the underlying resource.
func (r *Resource) Underlying() framework.Resource {
	return r
}

func (r *Resource) getDNSRecordSetsClient() (*dns.RecordSetsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "creating Azure DNS record sets client")
	}

	return azureClients.DNSRecordSetsClient, nil
}

func (r *Resource) getDNSZonesClient() (*dns.ZonesClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "creating Azure DNS zones client")
	}

	return azureClients.DNSZonesClient, nil
}
