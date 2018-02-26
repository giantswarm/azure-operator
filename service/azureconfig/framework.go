package azureconfig

import (
	"sync"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	gsclient "github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/giantswarm/operatorkit/informer"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1"
)

type FrameworkConfig struct {
	G8sClient    gsclient.Interface
	K8sClient    kubernetes.Interface
	K8sExtClient apiextensionsclient.Interface
	Logger       micrologger.Logger

	AzureConfig     client.AzureConfig
	ProjectName     string
	TemplateVersion string
}

type Framework struct {
	logger micrologger.Logger

	framework *framework.Framework
	bootOnce  sync.Once
}

func NewFramework(config FrameworkConfig) (*framework.Framework, error) {
	var err error

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

	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig.%s", err)
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.ProjectName must not be empty")
	}
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TemplateVersion must not be empty")
	}

	var v1ResourceSet []framework.Resource
	{
		c := v1.ResourceSetConfig{
			K8sClient:       config.K8sClient,
			K8sExtClient:    config.K8sExtClient,
			Logger:          config.Logger,
			AzureConfig:     config.AzureConfig,
			ProjectName:     config.ProjectName,
			TemplateVersion: config.TemplateVersion,
		}

		v1ResourceSet, err = v1.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var crdClient *k8scrdclient.CRDClient
	{
		c := k8scrdclient.Config{
			K8sExtClient: config.K8sExtClient,
			Logger:       config.Logger,
		}

		crdClient, err = k8scrdclient.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var newInformer *informer.Informer
	{
		c := informer.Config{
			ResyncPeriod: informer.DefaultResyncPeriod,
			Watcher:      config.G8sClient.ProviderV1alpha1().AzureConfigs(""),
		}

		newInformer, err = informer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var f *framework.Framework
	{
		c := framework.Config{
			CRD:            v1alpha1.NewAzureConfigCRD(),
			CRDClient:      crdClient,
			Informer:       newInformer,
			Logger:         config.Logger,
			ResourceRouter: framework.DefaultResourceRouter(v1ResourceSet),

			BackOffFactory: framework.DefaultConfig().BackOffFactory,
			InitCtxFunc:    framework.DefaultConfig().InitCtxFunc,
		}

		f, err = framework.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return f, nil
}
