package controller

import (
	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/flag"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type AzureClusterConfig struct {
	InstallationName string
	K8sClient        k8sclient.Interface
	Logger           micrologger.Logger

	Flag  *flag.Flag
	Viper *viper.Viper

	Azure          setting.Azure
	AzureConfig    client.AzureClientSetConfig
	ProjectName    string
	RegistryDomain string

	Ignition         setting.Ignition
	OIDC             setting.OIDC
	SSOPublicKey     string
	TemplateVersion  string
	VMSSCheckWorkers int
}

type AzureCluster struct {
	*controller.Controller
}

func NewAzureCluster(config AzureClusterConfig) (*AzureCluster, error) {
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

	var resourceSet *controller.ResourceSet
	{
		c := AzureClusterResourceSetConfig{
			CertsSearcher: certsSearcher,
			K8sClient:     config.K8sClient,
			Logger:        config.Logger,

			Flag:  config.Flag,
			Viper: config.Viper,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.AzureConfig,
			Ignition:                 config.Ignition,
			InstallationName:         config.InstallationName,
			ProjectName:              config.ProjectName,
			RegistryDomain:           config.RegistryDomain,
			OIDC:                     config.OIDC,
			SSOPublicKey:             config.SSOPublicKey,
			VMSSCheckWorkers:         config.VMSSCheckWorkers,
		}

		resourceSet, err = NewAzureClusterResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Name:      config.ProjectName,
			ResourceSets: []*controller.ResourceSet{
				resourceSet,
			},
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha3.AzureCluster)
			},
		}

		operatorkitController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c := &AzureCluster{
		Controller: operatorkitController,
	}

	return c, nil
}
