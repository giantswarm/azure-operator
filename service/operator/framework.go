package operator

import (
	"github.com/giantswarm/azure-operator/service/cloudconfig"
	"github.com/giantswarm/azure-operator/service/resource/deployment"
	"github.com/giantswarm/azure-operator/service/resource/dnsrecord"
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
			return nil, microerror.Maskf(err, "certificatetpr.NewService")
		}
	}

	var cloudConfig *cloudconfig.CloudConfig
	{
		cloudConfigConfig := cloudconfig.DefaultConfig()
		cloudConfigConfig.AzureConfig = config.AzureConfig
		cloudConfigConfig.Logger = config.Logger

		cloudConfig, err = cloudconfig.New(cloudConfigConfig)
		if err != nil {
			return nil, microerror.Maskf(err, "cloudconfig.New")
		}
	}

	var resourceGroupResource *resourcegroup.Resource
	{
		c := resourcegroup.DefaultConfig()
		c.AzureConfig = config.AzureConfig
		c.Logger = config.Logger

		resourceGroupResource, err = resourcegroup.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "resourcegroup.New")
		}
	}

	var deploymentResource *deployment.Resource
	{
		c := deployment.DefaultConfig()
		c.TemplateVersion = config.TemplateVersion
		c.AzureConfig = config.AzureConfig
		c.CertWatcher = certWatcher
		c.CloudConfig = cloudConfig
		c.Logger = config.Logger

		deploymentResource, err = deployment.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "deployment.New")
		}
	}

	var dnsrecordResource *dnsrecord.Resource
	{
		c := dnsrecord.DefaultConfig()
		c.AzureConfig = config.AzureConfig
		c.Logger = config.Logger

		dnsrecordResource, err = dnsrecord.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "dnsrecord.New")
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
			return nil, microerror.Maskf(err, "tpr.New")
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
			dnsrecordResource,
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
