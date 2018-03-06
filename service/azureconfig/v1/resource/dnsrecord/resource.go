package dnsrecord

import (
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-09-01/dns"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/azure-operator/client"
)

const (
	// Name is the identifier of the resource.
	Name = "dnsrecordv1"
)

type Config struct {
	AzureConfig client.AzureConfig
	Logger      micrologger.Logger
}

// Resource manages Azure resource groups.
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
		logger: config.Logger.With(
			"resource", Name,
		),
	}

	return r, nil
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
