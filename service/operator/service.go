package operator

import (
	"sync"

	gsclient "github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
)

// Config represents the configuration used to create an Operator service.
type Config struct {
	// Dependencies.

	G8sClient    gsclient.Interface
	K8sClient    kubernetes.Interface
	K8sExtClient apiextensionsclient.Interface
	Logger       micrologger.Logger

	// Settings.

	AzureConfig     client.AzureConfig
	TemplateVersion string
}

// DefaultConfig provides a default configuration to create a new operator
// service by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		G8sClient:    nil,
		K8sClient:    nil,
		K8sExtClient: nil,
		Logger:       nil,
		// Settings.
		AzureConfig: client.DefaultAzureConfig(),
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
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig.%s", err)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.G8sClient must not be empty")
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sClient must not be empty")
	}
	if config.K8sExtClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sExtClient must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	// Settings.
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TemplateVersion must not be empty")
	}

	operatorFramework, err := newFramework(config)
	if err != nil {
		return nil, microerror.Maskf(err, "newFramework")
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

func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		go s.framework.Boot()
	})
}
