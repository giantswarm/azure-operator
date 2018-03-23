package service

import (
	"context"
	"sync"

	"github.com/giantswarm/microendpoint/service/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8srestconfig"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/spf13/viper"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	gsclient "github.com/giantswarm/apiextensions/pkg/clientset/versioned"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/flag"
	"github.com/giantswarm/azure-operator/service/azureconfig"
	"github.com/giantswarm/azure-operator/service/azureconfig/setting"
	"github.com/giantswarm/azure-operator/service/healthz"
)

// Config represents the configuration used to create a new service.
type Config struct {
	Logger micrologger.Logger

	Flag  *flag.Flag
	Viper *viper.Viper

	Description string
	GitCommit   string
	ProjectName string
	Source      string
}

type Service struct {
	AzureConfigFramework *framework.Framework
	Healthz              *healthz.Service
	Version              *version.Service

	bootOnce sync.Once
}

// New creates a new configured service object.
func New(config Config) (*Service, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.Flag == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Flag must not be empty", config)
	}
	if config.Viper == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Viper must not be empty", config)
	}
	if config.Description == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Description must not be empty", config)
	}
	if config.GitCommit == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.GitCommit must not be empty", config)
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ProjectName must not be empty", config)
	}
	if config.Source == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Source must not be empty", config)
	}

	var err error

	azure := setting.Azure{
		HostCluster: setting.AzureHostCluster{
			CIDR:           config.Viper.GetString(config.Flag.Service.Azure.HostCluster.CIDR),
			ResourceGroup:  config.Viper.GetString(config.Flag.Service.Azure.HostCluster.ResourceGroup),
			VirtualNetwork: config.Viper.GetString(config.Flag.Service.Azure.HostCluster.VirtualNetwork),
		},
		Location: config.Viper.GetString(config.Flag.Service.Azure.Location),
	}

	azureConfig := client.AzureConfig{
		Logger: config.Logger,

		ClientID:       config.Viper.GetString(config.Flag.Service.Azure.ClientID),
		ClientSecret:   config.Viper.GetString(config.Flag.Service.Azure.ClientSecret),
		SubscriptionID: config.Viper.GetString(config.Flag.Service.Azure.SubscriptionID),
		TenantID:       config.Viper.GetString(config.Flag.Service.Azure.TenantID),
	}

	var restConfig *rest.Config
	{
		c := k8srestconfig.Config{
			Logger: config.Logger,

			Address:   config.Viper.GetString(config.Flag.Service.Kubernetes.Address),
			InCluster: config.Viper.GetBool(config.Flag.Service.Kubernetes.InCluster),
			TLS: k8srestconfig.TLSClientConfig{
				CAFile:  config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CAFile),
				CrtFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.CrtFile),
				KeyFile: config.Viper.GetString(config.Flag.Service.Kubernetes.TLS.KeyFile),
			},
		}

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

	var azureConfigFramework *framework.Framework
	{
		c := azureconfig.FrameworkConfig{
			G8sClient:    g8sClient,
			K8sClient:    k8sClient,
			K8sExtClient: k8sExtClient,
			Logger:       config.Logger,

			Azure:            azure,
			AzureConfig:      azureConfig,
			InstallationName: config.Viper.GetString(config.Flag.Service.Installation.Name),
			ProjectName:      config.ProjectName,
			TemplateVersion:  config.Viper.GetString(config.Flag.Service.Azure.Template.URI.Version),
		}

		azureConfigFramework, err = azureconfig.NewFramework(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var healthzService *healthz.Service
	{
		c := healthz.Config{
			AzureConfig: azureConfig,
			Logger:      config.Logger,
		}

		healthzService, err = healthz.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var versionService *version.Service
	{
		versionConfig := version.DefaultConfig()
		versionConfig.Description = config.Description
		versionConfig.GitCommit = config.GitCommit
		versionConfig.Name = config.ProjectName
		versionConfig.Source = config.Source
		versionConfig.VersionBundles = NewVersionBundles()

		versionService, err = version.New(versionConfig)
		if err != nil {
			return nil, microerror.Maskf(err, "version.New")
		}
	}

	newService := &Service{
		AzureConfigFramework: azureConfigFramework,
		Healthz:              healthzService,
		Version:              versionService,

		bootOnce: sync.Once{},
	}

	return newService, nil
}

func (s *Service) Boot(ctx context.Context) {
	s.bootOnce.Do(func() {
		s.AzureConfigFramework.Boot()
	})
}
