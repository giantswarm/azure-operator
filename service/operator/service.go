package operator

import (
	"sync"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
)

// Config represents the configuration used to create an Operator service.
type Config struct {
	// Dependencies.

	AzureConfig *client.AzureConfig
	K8sClient   kubernetes.Interface
	Resources   []framework.Resource

	// Settings.

	LoggerConfig micrologger.Config

	TemplateVersion string
}

// DefaultConfig provides a default configuration to create a new operator
// service by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		AzureConfig: nil,
		K8sClient:   nil,

		// Settings.

		LoggerConfig: micrologger.DefaultConfig(),
	}
}

// Service implements the Operator service interface.
type Service struct {
	// Internals.

	logger    micrologger.Logger
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

	// Settings.
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TemplateVersion must not be empty")
	}

	logger, err := micrologger.New(config.LoggerConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "creating logger")
	}

	operatorFramework, err := newFramework(config)
	if err != nil {
		return nil, microerror.Maskf(err, "creating operatorkit framework")
	}

	newService := &Service{
		// Dependencies.
		logger: logger,

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
