package operator

import (
	"fmt"
	"sync"

	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/tpr"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/giantswarm/azure-operator/flag"
)

// Config represents the configuration used to create an Operator service.
type Config struct {
	// Dependencies.
	Logger    micrologger.Logger
	K8sClient kubernetes.Interface

	// Settings.
	Flag  *flag.Flag
	Viper *viper.Viper
}

// DefaultConfig provides a default configuration to create a new operator
// service by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		K8sClient: nil,
		Logger:    nil,

		// Settings.
		Flag:  nil,
		Viper: nil,
	}
}

// Service implements the Operator service interface.
type Service struct {
	Config

	// Internals.
	bootOnce sync.Once
	tpr      *tpr.TPR
}

// New creates a new configured Operator service.
func New(config Config) (*Service, error) {
	// Dependencies.
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "kubernetes client must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}

	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "viper must not be empty")
	}

	tprConfig := tpr.Config{
		K8sClient:   config.K8sClient,
		Logger:      config.Logger,
		Name:        azuretpr.Name,
		Version:     azuretpr.VersionV1,
		Description: azuretpr.Description,
	}
	tpr, err := tpr.New(tprConfig)
	if err != nil {
		return nil, microerror.Maskf(err, "creating TPR for %#v", tprConfig)
	}

	newService := &Service{
		Config: config,

		// Internals
		bootOnce: sync.Once{},
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

		newResourceEventHandler := &cache.ResourceEventHandlerFuncs{
			AddFunc:    s.addFunc,
			DeleteFunc: s.deleteFunc,
		}
		newZeroObjectFactory := &tpr.ZeroObjectFactoryFuncs{
			NewObjectFunc:     func() runtime.Object { return &azuretpr.CustomObject{} },
			NewObjectListFunc: func() runtime.Object { return &azuretpr.List{} },
		}

		s.tpr.NewInformer(newResourceEventHandler, newZeroObjectFactory).Run(nil)
	})
}

func (s *Service) addFunc(obj interface{}) {
	customObject := *obj.(*azuretpr.CustomObject)
	s.Config.Logger.Log("debug", fmt.Sprintf("creating cluster '%s'", customObject.Spec.Cluster.Cluster.ID))

	// TODO Add stub code for creating an Azure Resource Group.
}

// deleteFunc TODO
func (s *Service) deleteFunc(obj interface{}) {
	customObject := *obj.(*azuretpr.CustomObject)
	s.Config.Logger.Log("debug", fmt.Sprintf("deleting cluster '%s'", customObject.Spec.Cluster.Cluster.ID))

	// TODO Add stub code for deleting the Azure Resource Group.
}
