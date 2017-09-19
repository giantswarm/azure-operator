package operator

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
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
	Backoff           backoff.BackOff
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
		Backoff:           nil,
		K8sClient:         nil,
		Logger:            nil,
		OperatorFramework: nil,

		// Settings.
		Flag:  nil,
		Viper: nil,
	}
}

// Service implements the Operator service interface.
type Service struct {
	Config

	// Dependencies.
	backoff           backoff.BackOff
	logger            micrologger.Logger
	operatorFramework *framework.Framework

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
	if config.Backoff == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.BackOff must not be empty")
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
		backoff:           config.Backoff,
		logger:            config.Logger,
		operatorFramework: config.OperatorFramework,

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
		o := func() error {
			err := s.bootWithError()
			if err != nil {
				return microerror.Mask(err)
			}

			return nil
		}

		n := func(err error, d time.Duration) {
			s.logger.Log("warning", fmt.Sprintf("retrying operator boot due to error: %#v", microerror.Mask(err)))
		}

		err := backoff.RetryNotify(o, s.backoff, n)
		if err != nil {
			s.logger.Log("error", fmt.Sprintf("stop operator boot retries due to too many errors: %#v", microerror.Mask(err)))
			os.Exit(1)
		}
	})
}

func (s *Service) bootWithError() error {
	err := s.tpr.CreateAndWait()
	if tpr.IsAlreadyExists(err) {
		s.Logger.Log("debug", "third party resource already exists")
	} else if err != nil {
		return microerror.Mask(err)
	}

	s.Logger.Log("debug", "starting list/watch")

	newResourceEventHandler := s.operatorFramework.NewCacheResourceEventHandler()

	newZeroObjectFactory := &tpr.ZeroObjectFactoryFuncs{
		NewObjectFunc:     func() runtime.Object { return &azuretpr.CustomObject{} },
		NewObjectListFunc: func() runtime.Object { return &azuretpr.List{} },
	}

	s.tpr.NewInformer(newResourceEventHandler, newZeroObjectFactory).Run(nil)

	return nil
}
