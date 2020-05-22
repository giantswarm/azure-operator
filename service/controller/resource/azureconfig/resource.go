package azureconfig

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/flag"
)

const (
	Name = "azureconfig"
)

type Config struct {
	Logger micrologger.Logger

	Flag  *flag.Flag
	Viper *viper.Viper

	CtrlClient client.Client
}

type Resource struct {
	logger micrologger.Logger

	flag  *flag.Flag
	viper *viper.Viper

	ctrlClient client.Client
}

func New(config Config) (*Resource, error) {
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Flag must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Viper must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}

	newResource := &Resource{
		logger: config.Logger,

		flag:  config.Flag,
		viper: config.Viper,

		ctrlClient: config.CtrlClient,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}
