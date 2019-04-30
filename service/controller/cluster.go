package controller

import (
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	gsclient "github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/client/k8scrdclient"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/informer"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v4patch1"
	"github.com/giantswarm/azure-operator/service/controller/v4patch2"
	"github.com/giantswarm/azure-operator/service/controller/v5"
	"github.com/giantswarm/azure-operator/service/controller/v6"
	"github.com/giantswarm/azure-operator/service/controller/v7"
	"github.com/giantswarm/azure-operator/service/controller/v8"
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
	IgnitionPath    string
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

	var certsSearcher *certs.Searcher
	{
		c := certs.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		certsSearcher, err = certs.NewSearcher(c)
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

	var v4Patch1ResourceSet *controller.ResourceSet
	{
		c := v4patch1.ResourceSetConfig{
			G8sClient: config.G8sClient,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			InstallationName:         config.InstallationName,
			ProjectName:              config.ProjectName,
			OIDC:                     config.OIDC,
			SSOPublicKey:             config.SSOPublicKey,
			TemplateVersion:          config.TemplateVersion,
		}

		v4Patch1ResourceSet, err = v4patch1.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v4Patch2ResourceSet *controller.ResourceSet
	{
		c := v4patch2.ResourceSetConfig{
			G8sClient: config.G8sClient,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			InstallationName:         config.InstallationName,
			ProjectName:              config.ProjectName,
			OIDC:                     config.OIDC,
			SSOPublicKey:             config.SSOPublicKey,
			TemplateVersion:          config.TemplateVersion,
		}

		v4Patch2ResourceSet, err = v4patch2.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v5ResourceSet *controller.ResourceSet
	{
		c := v5.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.G8sClient,
			K8sClient:     config.K8sClient,
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			IgnitionPath:             config.IgnitionPath,
			InstallationName:         config.InstallationName,
			ProjectName:              config.ProjectName,
			OIDC:                     config.OIDC,
			SSOPublicKey:             config.SSOPublicKey,
			TemplateVersion:          config.TemplateVersion,
		}

		v5ResourceSet, err = v5.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v6ResourceSet *controller.ResourceSet
	{
		c := v6.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.G8sClient,
			K8sClient:     config.K8sClient,
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			IgnitionPath:             config.IgnitionPath,
			InstallationName:         config.InstallationName,
			ProjectName:              config.ProjectName,
			OIDC:                     config.OIDC,
			SSOPublicKey:             config.SSOPublicKey,
			TemplateVersion:          config.TemplateVersion,
		}

		v6ResourceSet, err = v6.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v7ResourceSet *controller.ResourceSet
	{
		c := v7.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.G8sClient,
			K8sClient:     config.K8sClient,
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			IgnitionPath:             config.IgnitionPath,
			InstallationName:         config.InstallationName,
			ProjectName:              config.ProjectName,
			OIDC:                     config.OIDC,
			SSOPublicKey:             config.SSOPublicKey,
			TemplateVersion:          config.TemplateVersion,
		}

		v7ResourceSet, err = v7.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v8ResourceSet *controller.ResourceSet
	{
		c := v8.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.G8sClient,
			K8sClient:     config.K8sClient,
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			IgnitionPath:             config.IgnitionPath,
			InstallationName:         config.InstallationName,
			ProjectName:              config.ProjectName,
			OIDC:                     config.OIDC,
			SSOPublicKey:             config.SSOPublicKey,
			TemplateVersion:          config.TemplateVersion,
		}

		v8ResourceSet, err = v8.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			CRD:       v1alpha1.NewAzureConfigCRD(),
			CRDClient: crdClient,
			Informer:  newInformer,
			Logger:    config.Logger,
			ResourceSets: []*controller.ResourceSet{
				v4Patch1ResourceSet,
				v4Patch2ResourceSet,
				v5ResourceSet,
				v6ResourceSet,
				v7ResourceSet,
				v8ResourceSet,
			},
			RESTClient: config.G8sClient.ProviderV1alpha1().RESTClient(),

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
