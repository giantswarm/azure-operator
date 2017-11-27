package operator

import (
	"github.com/giantswarm/azure-operator/service/cloudconfig"
	"github.com/giantswarm/azure-operator/service/resource/deployment"
	"github.com/giantswarm/azure-operator/service/resource/resourcegroup"
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/certificatetpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/giantswarm/operatorkit/informer"
	"github.com/giantswarm/operatorkit/tpr"
	"k8s.io/apimachinery/pkg/runtime"
)

func newFramework(config Config) (*framework.Framework, error) {
	var err error

	var certWatcher certificatetpr.Searcher
	{
		certConfig := certificatetpr.DefaultServiceConfig()
		certConfig.K8sClient = config.K8sClient
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
		resourceGroupConfig.AzureConfig = config.AzureConfig
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
		deploymentConfig.AzureConfig = config.AzureConfig
		deploymentConfig.CertWatcher = certWatcher
		deploymentConfig.CloudConfig = cloudConfigService
		deploymentConfig.Logger = config.Logger

		deploymentResource, err = deployment.New(deploymentConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var newTPR *tpr.TPR
	{
		c := tpr.DefaultConfig()

		c.K8sClient = config.K8sClient
		c.Logger = config.Logger

		c.Description = azuretpr.Description
		c.Name = azuretpr.Name
		c.Version = azuretpr.VersionV1

		newTPR, err = tpr.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var newWatcherFactory informer.WatcherFactory
	{

		zeroObjectFactory := &tpr.ZeroObjectFactoryFuncs{
			NewObjectFunc:     func() runtime.Object { return &azuretpr.CustomObject{} },
			NewObjectListFunc: func() runtime.Object { return &azuretpr.List{} },
		}

		newWatcherFactory = informer.NewWatcherFactory(config.K8sClient.Discovery().RESTClient(), newTPR.WatchEndpoint(""), zeroObjectFactory)
	}

	var newInformer *informer.Informer
	{
		c := informer.DefaultConfig()

		c.WatcherFactory = newWatcherFactory

		newInformer, err = informer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var f *framework.Framework
	{
		resources := []framework.Resource{
			resourceGroupResource,
			deploymentResource,
		}

		c := framework.DefaultConfig()

		c.Logger = config.Logger
		c.ResourceRouter = framework.DefaultResourceRouter(resources)
		c.Informer = newInformer
		c.TPR = newTPR

		f, err = framework.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return f, nil
}
