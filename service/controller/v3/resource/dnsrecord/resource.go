package dnsrecord

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-09-01/dns"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v3/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "dnsrecordv3"
)

type Config struct {
	HostAzureConfig client.AzureClientSetConfig
	Logger          micrologger.Logger
}

// Resource manages Azure resource groups.
type Resource struct {
	hostAzureConfig client.AzureClientSetConfig
	logger          micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if err := config.HostAzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.HostAzureConfig.%s", err)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	r := &Resource{
		hostAzureConfig: config.HostAzureConfig,
		logger:          config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getDNSRecordSetsClient() (*dns.RecordSetsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.DNSRecordSetsClient, nil
}

func (r *Resource) getDNSZonesClient(ctx context.Context) (*dns.ZonesClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.DNSZonesClient, nil
}
