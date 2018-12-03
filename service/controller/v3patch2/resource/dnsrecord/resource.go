package dnsrecord

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v3patch2/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "dnsrecordv3patch2"
)

type Config struct {
	HostAzureClientSetConfig client.AzureClientSetConfig
	Logger                   micrologger.Logger
}

// Resource manages Azure resource groups.
type Resource struct {
	hostAzureClientSetConfig client.AzureClientSetConfig
	logger                   micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if err := config.HostAzureClientSetConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.HostAzureClientSetConfig.%s", err)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	r := &Resource{
		hostAzureClientSetConfig: config.HostAzureClientSetConfig,
		logger:                   config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getDNSRecordSetsHostClient() (*dns.RecordSetsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.hostAzureClientSetConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.DNSRecordSetsClient, nil
}

func (r *Resource) getDNSRecordSetsGuestClient(ctx context.Context) (*dns.RecordSetsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.DNSRecordSetsClient, nil
}

func (r *Resource) getDNSZonesGuestClient(ctx context.Context) (*dns.ZonesClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.DNSZonesClient, nil
}
