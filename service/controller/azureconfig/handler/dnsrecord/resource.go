package dnsrecord

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
)

const (
	// Name is the identifier of the resource.
	Name = "dnsrecord"
)

type Config struct {
	CPRecordSetsClient dns.RecordSetsClient
	Logger             micrologger.Logger
}

// Resource manages Azure resource groups.
type Resource struct {
	cpRecordSetsClient dns.RecordSetsClient
	logger             micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	r := &Resource{
		cpRecordSetsClient: config.CPRecordSetsClient,
		logger:             config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getDNSRecordSetsGuestClient(ctx context.Context) (*dns.RecordSetsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.DNSRecordSetsClient, nil
}

func (r *Resource) getDNSZonesGuestClient(ctx context.Context) (*dns.ZonesClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.DNSZonesClient, nil
}
