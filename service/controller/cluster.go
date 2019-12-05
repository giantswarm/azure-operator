package controller

import (
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/k8sclient/k8scrdclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/informer"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	v10 "github.com/giantswarm/azure-operator/service/controller/v10"
	v10patch1 "github.com/giantswarm/azure-operator/service/controller/v10patch1"
	v10patch2 "github.com/giantswarm/azure-operator/service/controller/v10patch2"
	v11 "github.com/giantswarm/azure-operator/service/controller/v11"
	v12 "github.com/giantswarm/azure-operator/service/controller/v12"
	v6 "github.com/giantswarm/azure-operator/service/controller/v6"
	v7 "github.com/giantswarm/azure-operator/service/controller/v7"
	v8 "github.com/giantswarm/azure-operator/service/controller/v8"
	"github.com/giantswarm/azure-operator/service/controller/v8patch1"
	v9 "github.com/giantswarm/azure-operator/service/controller/v9"
)

type ClusterConfig struct {
	InstallationName string
	K8sClient        k8sclient.Interface
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
	var err error

	var certsSearcher *certs.Searcher
	{
		c := certs.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		certsSearcher, err = certs.NewSearcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var newInformer *informer.Informer
	{
		c := informer.Config{
			Logger:  config.Logger,
			Watcher: config.K8sClient.G8sClient().ProviderV1alpha1().AzureConfigs(""),

			RateWait:     informer.DefaultRateWait,
			ResyncPeriod: 3 * time.Minute,
		}

		newInformer, err = informer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v6ResourceSet *controller.ResourceSet
	{
		c := v6.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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

	var v8patch1ResourceSet *controller.ResourceSet
	{
		c := v8patch1.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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

		v8patch1ResourceSet, err = v8patch1.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v9ResourceSet *controller.ResourceSet
	{
		c := v9.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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

		v9ResourceSet, err = v9.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v10ResourceSet *controller.ResourceSet
	{
		c := v10.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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

		v10ResourceSet, err = v10.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v10patch1ResourceSet *controller.ResourceSet
	{
		c := v10patch1.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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

		v10patch1ResourceSet, err = v10patch1.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v10patch2ResourceSet *controller.ResourceSet
	{
		c := v10patch2.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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

		v10patch2ResourceSet, err = v10patch2.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v11ResourceSet *controller.ResourceSet
	{
		c := v11.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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

		v11ResourceSet, err = v11.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v12ResourceSet *controller.ResourceSet
	{
		c := v12.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
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

		v12ResourceSet, err = v12.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			CRD:       v1alpha1.NewAzureConfigCRD(),
			CRDClient: config.K8sClient.CRDClient().(*k8scrdclient.CRDClient),
			Informer:  newInformer,
			Logger:    config.Logger,
			ResourceSets: []*controller.ResourceSet{
				v6ResourceSet,
				v7ResourceSet,
				v8ResourceSet,
				v8patch1ResourceSet,
				v9ResourceSet,
				v10ResourceSet,
				v10patch1ResourceSet,
				v10patch2ResourceSet,
				v11ResourceSet,
				v12ResourceSet,
			},
			RESTClient: config.K8sClient.G8sClient().ProviderV1alpha1().RESTClient(),

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
