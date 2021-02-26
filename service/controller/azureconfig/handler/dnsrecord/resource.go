package dnsrecord

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsaware"
)

const (
	// Name is the identifier of the resource.
	Name = "dnsrecord"
)

type Config struct {
	Logger               micrologger.Logger
	MCAzureClientFactory credentialsaware.Factory
	WCAzureClientFactory credentialsaware.Factory
}

// Resource manages Azure resource groups.
type Resource struct {
	logger               micrologger.Logger
	mcAzureClientFactory credentialsaware.Factory
	wcAzureClientFactory credentialsaware.Factory
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}
	if config.MCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.WCAzureClientFactory must not be empty")
	}
	if config.WCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.WCAzureClientFactory must not be empty")
	}

	r := &Resource{
		mcAzureClientFactory: config.MCAzureClientFactory,
		wcAzureClientFactory: config.WCAzureClientFactory,
		logger:               config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
