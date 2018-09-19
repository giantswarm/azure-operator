package vpngateway

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

const (
	Name = "vpngatewayv5"
)

// Config is the configuration required by Resource.
type Config struct {
	Logger micrologger.Logger

	Azure                    setting.Azure
	HostAzureClientSetConfig client.AzureClientSetConfig
}

// Resource manages Azure virtual network peering.
type Resource struct {
	logger micrologger.Logger

	azure                    setting.Azure
	hostAzureClientSetConfig client.AzureClientSetConfig
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.HostAzureClientSetConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HostAzureClientSetConfig.%s", config, err)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	r := &Resource{
		logger: config.Logger,

		azure: config.Azure,
		hostAzureClientSetConfig: config.HostAzureClientSetConfig,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
