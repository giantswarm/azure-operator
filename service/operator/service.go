package operator

import (
	"fmt"
	"sync"

	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/giantswarm/operatorkit/tpr"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/flag"
)

// Config represents the configuration used to create an Operator service.
type Config struct {
	// Dependencies.

	AzureConfig       *client.AzureConfig
	Logger            micrologger.Logger
	OperatorFramework *framework.Framework
	K8sClient         kubernetes.Interface
	Resources         []framework.Resource

	// Settings.

	Flag  *flag.Flag
	Viper *viper.Viper
}

// DefaultConfig provides a default configuration to create a new operator
// service by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		AzureConfig:       nil,
		K8sClient:         nil,
		Logger:            nil,
		OperatorFramework: nil,
		Resources:         nil,

		// Settings.
		Flag:  nil,
		Viper: nil,
	}
}

// Service implements the Operator service interface.
type Service struct {
	Config

	// Dependencies.
	logger            micrologger.Logger
	operatorFramework *framework.Framework
	resources         []framework.Resource

	// Internals.

	bootOnce sync.Once
	tpr      *tpr.TPR
	mutex    sync.Mutex
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
	if config.OperatorFramework == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.OperatorFramework must not be empty")
	}
	if config.Resources == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Resources must not be empty")
	}

	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Viper must not be empty")
	}

	tprConfig := tpr.DefaultConfig()
	tprConfig.K8sClient = config.K8sClient
	tprConfig.Logger = config.Logger
	tprConfig.Name = azuretpr.Name
	tprConfig.Version = azuretpr.VersionV1
	tprConfig.Description = azuretpr.Description

	tpr, err := tpr.New(tprConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "creating TPR for %#v", tprConfig)
	}

	newService := &Service{
		Config: config,

		// Dependencies.
		logger:            config.Logger,
		operatorFramework: config.OperatorFramework,
		resources:         config.Resources,

		// Internals.
		bootOnce: sync.Once{},
		mutex:    sync.Mutex{},
		tpr:      tpr,
	}

	return newService, nil
}

// Boot starts the service and implements the watch for azuretpr.
func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		err := s.tpr.CreateAndWait()
		if tpr.IsAlreadyExists(err) {
			s.Logger.Log("debug", "third party resource already exists")
		} else if err != nil {
			s.Logger.Log("error", fmt.Sprintf("%#v", err))
			return
		}

		s.Logger.Log("debug", "starting list/watch")

		newResourceEventHandler := s.operatorFramework.NewCacheResourceEventHandler()

		newZeroObjectFactory := &tpr.ZeroObjectFactoryFuncs{
			NewObjectFunc:     func() runtime.Object { return &azuretpr.CustomObject{} },
			NewObjectListFunc: func() runtime.Object { return &azuretpr.List{} },
		}

		s.tpr.NewInformer(newResourceEventHandler, newZeroObjectFactory).Run(nil)
	})
}
