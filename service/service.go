package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/giantswarm/certificatetpr"
	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8sclient"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/flag"
	"github.com/giantswarm/azure-operator/service/cloudconfig"
	"github.com/giantswarm/azure-operator/service/operator"
	"github.com/giantswarm/azure-operator/service/resource/deployment"
	"github.com/giantswarm/azure-operator/service/resource/resourcegroup"
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
	config.Logger.Log("debug", fmt.Sprintf("creating azure-operator with config: %#v", config))

	// Settings.
	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Flag must not be empty")
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Viper must not be empty")
	}

	var err error

	var azureConfig *client.AzureConfig
	{
		azureConfig = client.DefaultAzureConfig()
		azureConfig.ClientID = config.Viper.GetString(config.Flag.Service.Azure.ClientID)
		azureConfig.ClientSecret = config.Viper.GetString(config.Flag.Service.Azure.ClientSecret)
		azureConfig.SubscriptionID = config.Viper.GetString(config.Flag.Service.Azure.SubscriptionID)
		azureConfig.TenantID = config.Viper.GetString(config.Flag.Service.Azure.TenantID)
	}

	var k8sClient kubernetes.Interface
	{
		k8sConfig := k8sclient.DefaultConfig()
		k8sConfig.Address = config.Viper.GetString(config.Flag.Service.Kubernetes.Address)
		k8sConfig.Logger = config.Logger
		k8sConfig.InCluster = config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster)
		k8sConfig.TLS.CAFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile)
		k8sConfig.TLS.CrtFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile)
		k8sConfig.TLS.KeyFile = config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile)

		k8sClient, err = k8sclient.New(k8sConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var certWatcher certificatetpr.Searcher
	{
		certConfig := certificatetpr.DefaultServiceConfig()
		certConfig.K8sClient = k8sClient
		certConfig.Logger = config.Logger
		certWatcher, err = certificatetpr.NewService(certConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var cloudConfigService *cloudconfig.CloudConfig
	{
		cloudConfigConfig := cloudconfig.DefaultConfig()
		cloudConfigConfig.Flag = config.Flag
		cloudConfigConfig.Logger = config.Logger
		cloudConfigConfig.Viper = config.Viper

		cloudConfigService, err = cloudconfig.New(cloudConfigConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resourceGroupResource framework.Resource
	{
		resourceGroupConfig := resourcegroup.DefaultConfig()
		resourceGroupConfig.AzureConfig = azureConfig
		resourceGroupConfig.Logger = config.Logger

		resourceGroupResource, err = resourcegroup.New(resourceGroupConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var deploymentResource framework.Resource
	{
		deploymentConfig := deployment.DefaultConfig()
		deploymentConfig.TemplateVersion = config.Viper.GetString(config.Flag.Service.Azure.Template.URI.Version)
		deploymentConfig.AzureConfig = azureConfig
		deploymentConfig.CertWatcher = certWatcher
		deploymentConfig.CloudConfig = cloudConfigService
		deploymentConfig.Logger = config.Logger

		deploymentResource, err = deployment.New(deploymentConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorFramework *framework.Framework
	{
		resources := []framework.Resource{
			resourceGroupResource,
			deploymentResource,
		}

		frameworkConfig := framework.DefaultConfig()

		frameworkConfig.Logger = config.Logger
		frameworkConfig.ResourceRouter = framework.DefaultResourceRouter(resources)

		operatorFramework, err = framework.New(frameworkConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorBackOff *backoff.ExponentialBackOff
	{
		operatorBackOff = backoff.NewExponentialBackOff()
		operatorBackOff.MaxElapsedTime = 5 * time.Minute
	}

	var operatorService *operator.Service
	{
		operatorConfig := operator.DefaultConfig()
		operatorConfig.AzureConfig = azureConfig
		operatorConfig.Backoff = operatorBackOff
		operatorConfig.Flag = config.Flag
		operatorConfig.K8sClient = k8sClient
		operatorConfig.Logger = config.Logger
		operatorConfig.OperatorFramework = operatorFramework
		operatorConfig.Viper = config.Viper

		operatorService, err = operator.New(operatorConfig)
		if err != nil {
			return nil, microerror.Mask(err)
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
			return nil, microerror.Mask(err)
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
