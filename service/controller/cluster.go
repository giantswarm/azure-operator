package controller

import (
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	gsclient "github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/informer"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v1"
	"github.com/giantswarm/azure-operator/service/controller/v2"
	"github.com/giantswarm/azure-operator/service/controller/v3"
)

type ClusterConfig struct {
	G8sClient        gsclient.Interface
	InstallationName string
	K8sClient        kubernetes.Interface
	K8sExtClient     apiextensionsclient.Interface
	Logger           micrologger.Logger

	Azure           setting.Azure
	AzureConfig     client.AzureClientSetConfig
	ProjectName     string
	OIDC            setting.OIDC
	SSOPublicKey    string
	TemplateVersion string
}

type Cluster struct {
	*controller.Controller
}

func NewCluster(config ClusterConfig) (*Cluster, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}

	var err error

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
			Logger:  config.Logger,
			Watcher: config.G8sClient.ProviderV1alpha1().AzureConfigs(""),

			RateWait:     informer.DefaultRateWait,
			ResyncPeriod: 3 * time.Minute,
		}

		newInformer, err = informer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v1ResourceSet *controller.ResourceSet
	{
		c := v1.ResourceSetConfig{
			K8sClient:    config.K8sClient,
			K8sExtClient: config.K8sExtClient,
			Logger:       config.Logger,

			Azure:            config.Azure,
			AzureConfig:      config.AzureConfig,
			InstallationName: config.InstallationName,
			ProjectName:      config.ProjectName,
			TemplateVersion:  config.TemplateVersion,
		}

		v1ResourceSet, err = v1.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v2ResourceSet *controller.ResourceSet
	{
		c := v2.ResourceSetConfig{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			Azure:            config.Azure,
			AzureConfig:      config.AzureConfig,
			InstallationName: config.InstallationName,
			ProjectName:      config.ProjectName,
			OIDC:             config.OIDC,
			TemplateVersion:  config.TemplateVersion,
		}

		v2ResourceSet, err = v2.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v3ResourceSet *controller.ResourceSet
	{
		c := v3.ResourceSetConfig{
			G8sClient: config.G8sClient,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			Azure:            config.Azure,
			HostAzureConfig:  config.AzureConfig,
			InstallationName: config.InstallationName,
			ProjectName:      config.ProjectName,
			OIDC:             config.OIDC,
			SSOPublicKey:     config.SSOPublicKey,
			TemplateVersion:  config.TemplateVersion,
		}

		v3ResourceSet, err = v3.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resourceRouter *controller.ResourceRouter
	{
		c := controller.ResourceRouterConfig{
			Logger: config.Logger,

			ResourceSets: []*controller.ResourceSet{
				v1ResourceSet,
				v2ResourceSet,
				v3ResourceSet,
			},
		}

		resourceRouter, err = controller.NewResourceRouter(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			CRD:            v1alpha1.NewAzureConfigCRD(),
			CRDClient:      crdClient,
			Informer:       newInformer,
			Logger:         config.Logger,
			ResourceRouter: resourceRouter,
			RESTClient:     config.G8sClient.ProviderV1alpha1().RESTClient(),

			Name: config.ProjectName,
		}

		operatorkitController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &Cluster{
		Controller: operatorkitController,
	}

	return c, nil
}
