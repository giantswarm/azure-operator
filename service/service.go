package service

import (
	"fmt"
	"sync"

	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8srestconfig"
	"github.com/spf13/viper"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	gsclient "github.com/giantswarm/apiextensions/pkg/clientset/versioned"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/flag"
	"github.com/giantswarm/azure-operator/service/operator"
)

// Config represents the configuration used to create a new service.
type Config struct {
	// Dependencies.

	Logger micrologger.Logger

	// Settings.

	Flag  *flag.Flag
	Viper *viper.Viper

	Description string
	GitCommit   string
	Name        string
	Source      string
}

// DefaultConfig provides a default configuration to create a new service by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		Logger: nil,

		// Settings.
		Flag:  nil,
		Viper: nil,

		Description: "",
		GitCommit:   "",
		Name:        "",
		Source:      "",
	}
}

// New creates a new configured service object.
func New(config Config) (*Service, error) {
	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}
	config.Logger.Log("debug", fmt.Sprintf("creating azure-operator gitCommit:%s", config.GitCommit))

	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Viper must not be empty")
	}

	var err error

	var azureConfig client.AzureConfig
	{
		azureConfig = client.DefaultAzureConfig()
		azureConfig.Logger = config.Logger
		azureConfig.ClientID = config.Viper.GetString(config.Flag.Service.Azure.ClientID)
		azureConfig.ClientSecret = config.Viper.GetString(config.Flag.Service.Azure.ClientSecret)
		azureConfig.SubscriptionID = config.Viper.GetString(config.Flag.Service.Azure.SubscriptionID)
		azureConfig.TenantID = config.Viper.GetString(config.Flag.Service.Azure.TenantID)
	}

	var restConfig *rest.Config
	{
		c := k8srestconfig.DefaultConfig()

		c.Logger = config.Logger

		c.Address = config.Viper.GetString(config.Flag.Service.Kubernetes.Address)
		c.InCluster = config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster)
		c.TLS.CAFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile)
		c.TLS.CrtFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile)
		c.TLS.KeyFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile)

		restConfig, err = k8srestconfig.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	g8sClient, err := gsclient.NewForConfig(restConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	k8sExtClient, err := apiextensionsclient.NewForConfig(restConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var operatorService *operator.Service
	{
		operatorConfig := operator.DefaultConfig()
		operatorConfig.Logger = config.Logger
		operatorConfig.AzureConfig = azureConfig
		operatorConfig.G8sClient = g8sClient
		operatorConfig.K8sClient = k8sClient
		operatorConfig.K8sExtClient = k8sExtClient
		operatorConfig.TemplateVersion = config.Viper.GetString(config.Flag.Service.Azure.Template.URI.Version)

		operatorService, err = operator.New(operatorConfig)
		if err != nil {
			return nil, microerror.Maskf(err, "operator.New")
		}
	}

	var versionService *version.Service
	{
		versionConfig := version.DefaultConfig()
		versionConfig.Description = config.Description
		versionConfig.GitCommit = config.GitCommit
		versionConfig.Name = config.Name
		versionConfig.Source = config.Source

		versionService, err = version.New(versionConfig)
		if err != nil {
			return nil, microerror.Maskf(err, "version.New")
		}
	}

	newService := &Service{
		// Dependencies.
		Operator: operatorService,
		Version:  versionService,

		// Internals
		bootOnce: sync.Once{},
	}

	return newService, nil
}

type Service struct {
	// Dependencies.
	Operator *operator.Service
	Version  *version.Service

	// Internals.
	bootOnce sync.Once
}

func (s *Service) Boot() {
	s.bootOnce.Do(func() {
		s.Operator.Boot()
	})
}
