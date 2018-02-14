package operator

import (
	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/cloudconfig"
	"github.com/giantswarm/azure-operator/service/resource/deployment"
	"github.com/giantswarm/azure-operator/service/resource/dnsrecord"
	"github.com/giantswarm/azure-operator/service/resource/resourcegroup"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/giantswarm/operatorkit/informer"
	"github.com/giantswarm/randomkeys"
)

func newFramework(config Config) (*framework.Framework, error) {
	var err error

	var crdClient *k8scrdclient.CRDClient
	{
		c := k8scrdclient.DefaultConfig()

		c.K8sExtClient = config.K8sExtClient
		c.Logger = config.Logger

		crdClient, err = k8scrdclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var certsSearcher *certs.Searcher
	{
		c := certs.DefaultConfig()

		c.K8sClient = config.K8sClient
		c.Logger = config.Logger

		certsSearcher, err = certs.NewSearcher(c)
		if err != nil {
			return nil, microerror.Maskf(err, "certs.NewSearcher")
		}
	}

	var randomkeysSearcher *randomkeys.Searcher
	{
		c := randomkeys.DefaultConfig()
		c.K8sClient = config.K8sClient
		c.Logger = config.Logger

		randomkeysSearcher, err = randomkeys.NewSearcher(c)
		if err != nil {
			return nil, microerror.Maskf(err, "randomkeys.NewSearcher")
		}
	}

	var cloudConfig *cloudconfig.CloudConfig
	{
		c := cloudconfig.DefaultConfig()

		c.AzureConfig = config.AzureConfig
		c.Logger = config.Logger
		c.RandomkeysSearcher = randomkeysSearcher

		cloudConfig, err = cloudconfig.New(c)
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

		c.CertsSearcher = certsSearcher
		c.Logger = config.Logger

		c.AzureConfig = config.AzureConfig
		c.CloudConfig = cloudConfig
		c.TemplateVersion = config.TemplateVersion

		deploymentResource, err = deployment.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "deployment.New")
		}
	}

	var dnsrecordResource *dnsrecord.Resource
	{
		c := dnsrecord.DefaultConfig()

		c.Logger = config.Logger

		c.AzureConfig = config.AzureConfig

		dnsrecordResource, err = dnsrecord.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "dnsrecord.New")
		}
	}

	var newInformer *informer.Informer
	{
		c := informer.DefaultConfig()

		c.Watcher = config.G8sClient.ProviderV1alpha1().AzureConfigs("")

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

		c.CRD = v1alpha1.NewAzureConfigCRD()
		c.CRDClient = crdClient
		c.Logger = config.Logger
		c.ResourceRouter = framework.DefaultResourceRouter(resources)
		c.Informer = newInformer

		f, err = framework.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return f, nil
}
