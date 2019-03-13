package cloudconfig

import (
	"github.com/giantswarm/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/randomkeys"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/network"
)

const (
	CertFilePermission = 0400
	FileOwnerUser      = "root"
	FileOwnerGroup     = "root"
	FilePermission     = 0700
)

type Config struct {
	CertsSearcher      certs.Interface
	Logger             micrologger.Logger
	RandomkeysSearcher randomkeys.Interface

	Azure setting.Azure
	// TODO(pk) remove as soon as we sort calico in Azure provider.
	AzureConfig  client.AzureClientSetConfig
	AzureNetwork network.Subnets
	IgnitionPath string
	OIDC         setting.OIDC
	SSOPublicKey string
}

type CloudConfig struct {
	logger             micrologger.Logger
	randomkeysSearcher randomkeys.Interface

	azure        setting.Azure
	azureConfig  client.AzureClientSetConfig
	azureNetwork network.Subnets
	ignitionPath string
	OIDC         setting.OIDC
	ssoPublicKey string
}

func New(config Config) (*CloudConfig, error) {
	if config.IgnitionPath == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.IgnitionPath must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.RandomkeysSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.RandomkeysSearcher must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureConfig.%s", config, err)
	}

	c := &CloudConfig{
		logger:             config.Logger,
		randomkeysSearcher: config.RandomkeysSearcher,

		azure:        config.Azure,
		azureConfig:  config.AzureConfig,
		azureNetwork: config.AzureNetwork,
		ignitionPath: config.IgnitionPath,
		OIDC:         config.OIDC,
		ssoPublicKey: config.SSOPublicKey,
	}

	return c, nil
}
