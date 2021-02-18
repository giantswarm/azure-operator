package vpnconnection

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"
)

const (
	Name = "vpnconnection"
)

// Config is the configuration required by Resource.
type Config struct {
	Azure                setting.Azure
	Logger               micrologger.Logger
	MCAzureClientFactory client.CredentialsAwareClientFactoryInterface
	WCAzureClientFactory client.CredentialsAwareClientFactoryInterface
}

// Resource manages Azure virtual network peering.
type Resource struct {
	azure                setting.Azure
	logger               micrologger.Logger
	mcAzureClientFactory client.CredentialsAwareClientFactoryInterface
	wcAzureClientFactory client.CredentialsAwareClientFactoryInterface
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.MCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}
	if config.WCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	r := &Resource{
		azure:                config.Azure,
		mcAzureClientFactory: config.MCAzureClientFactory,
		wcAzureClientFactory: config.WCAzureClientFactory,
		logger:               config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
