package dnsrecord

import (
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
)

const (
	// Name is the identifier of the resource.
	Name = "dnsrecordv2"
)

type Config struct {
	AzureConfig client.AzureClientSetConfig
	Logger      micrologger.Logger
}

// Resource manages Azure resource groups.
type Resource struct {
	azureConfig client.AzureClientSetConfig
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

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getDNSRecordSetsClient() (*dns.RecordSetsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.DNSRecordSetsClient, nil
}

func (r *Resource) getDNSZonesClient() (*dns.ZonesClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.DNSZonesClient, nil
}
