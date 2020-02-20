package controller

import (
	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	v10patch1 "github.com/giantswarm/azure-operator/service/controller/v10patch1"
	v11 "github.com/giantswarm/azure-operator/service/controller/v11"
	v12 "github.com/giantswarm/azure-operator/service/controller/v12"
	v13 "github.com/giantswarm/azure-operator/service/controller/v13"
	v14 "github.com/giantswarm/azure-operator/service/controller/v14"
	v7 "github.com/giantswarm/azure-operator/service/controller/v7"
)

type ClusterConfig struct {
	InstallationName string
	K8sClient        k8sclient.Interface
	Logger           micrologger.Logger

	Azure           setting.Azure
	AzureConfig     client.AzureClientSetConfig
	ProjectName     string
	Ignition        setting.Ignition
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

	var v7ResourceSet *controller.ResourceSet
	{
		c := v7.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			IgnitionPath:             config.Ignition.Path,
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

	var v10patch1ResourceSet *controller.ResourceSet
	{
		c := v10patch1.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			IgnitionPath:             config.Ignition.Path,
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

	var v11ResourceSet *controller.ResourceSet
	{
		c := v11.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			G8sClient:     config.K8sClient.G8sClient(),
			K8sClient:     config.K8sClient.K8sClient(),
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			IgnitionPath:             config.Ignition.Path,
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
			K8sClient:     config.K8sClient,
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			IgnitionPath:             config.Ignition.Path,
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

	var v13ResourceSet *controller.ResourceSet
	{
		c := v13.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			K8sClient:     config.K8sClient,
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			Ignition:                 config.Ignition,
			InstallationName:         config.InstallationName,
			ProjectName:              config.ProjectName,
			OIDC:                     config.OIDC,
			SSOPublicKey:             config.SSOPublicKey,
			TemplateVersion:          config.TemplateVersion,
		}

		v13ResourceSet, err = v13.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var v14ResourceSet *controller.ResourceSet
	{
		c := v14.ResourceSetConfig{
			CertsSearcher: certsSearcher,
			K8sClient:     config.K8sClient,
			Logger:        config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			Ignition:                 config.Ignition,
			InstallationName:         config.InstallationName,
			ProjectName:              config.ProjectName,
			OIDC:                     config.OIDC,
			SSOPublicKey:             config.SSOPublicKey,
			TemplateVersion:          config.TemplateVersion,
		}

		v14ResourceSet, err = v14.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			CRD:       v1alpha1.NewAzureConfigCRD(),
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Name:      config.ProjectName,
			ResourceSets: []*controller.ResourceSet{
				v7ResourceSet,
				v10patch1ResourceSet,
				v11ResourceSet,
				v12ResourceSet,
				v13ResourceSet,
				v14ResourceSet,
			},
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha1.AzureConfig)
			},
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
