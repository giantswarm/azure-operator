package operator

import (
	"sync"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/flag"
)

// Config represents the configuration used to create an Operator service.
type Config struct {
	// Dependencies.

	AzureConfig *client.AzureConfig
	Logger      micrologger.Logger
	K8sClient   kubernetes.Interface
	Resources   []framework.Resource

	// Settings.

	Flag  *flag.Flag
	Viper *viper.Viper
}

// DefaultConfig provides a default configuration to create a new operator
// service by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		AzureConfig: nil,
		K8sClient:   nil,
		Logger:      nil,

		// Settings.
		Flag:  nil,
		Viper: nil,
	}
}

// Service implements the Operator service interface.
type Service struct {
	// Dependencies.

	logger micrologger.Logger

	// Internals.

	framework *framework.Framework
	bootOnce  sync.Once
}

// New creates a new configured Operator service.
func New(config Config) (*Service, error) {
	// Dependencies.
	if config.AzureConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig must not be empty")
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sClient must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Viper must not be empty")
	}

	operatorFramework, err := newFramework(config)
	if err != nil {
		return nil, microerror.Maskf(err, "creating operatorkit framework")
	}

	newService := &Service{
		// Dependencies.
		logger: config.Logger,

		// Internals.
		framework: operatorFramework,
		bootOnce:  sync.Once{},
	}

	return newService, nil
}

// Boot starts the service and implements the watch for azuretpr.
func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		go s.framework.Boot()
	})
}
