package controller

import (
	"net"

	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/locker"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type ClusterConfig struct {
	InstallationName string
	K8sClient        k8sclient.Interface
	Locker           locker.Interface
	Logger           micrologger.Logger

	Azure          setting.Azure
	AzureConfig    client.AzureClientSetConfig
	ProjectName    string
	RegistryDomain string

	GuestPrivateSubnetMaskBits int
	GuestPublicSubnetMaskBits  int
	GuestSubnetMaskBits        int

	Ignition         setting.Ignition
	IPAMNetworkRange net.IPNet
	OIDC             setting.OIDC
	SSOPublicKey     string
	TemplateVersion  string
	VMSSCheckWorkers int
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

	var resourceSet *controller.ResourceSet
	{
		c := ResourceSetConfig{
			CertsSearcher: certsSearcher,
			K8sClient:     config.K8sClient,
			Locker:        config.Locker,
			Logger:        config.Logger,

			Azure:                      config.Azure,
			GuestPrivateSubnetMaskBits: config.GuestPrivateSubnetMaskBits,
			GuestPublicSubnetMaskBits:  config.GuestPublicSubnetMaskBits,
			GuestSubnetMaskBits:        config.GuestSubnetMaskBits,
			HostAzureClientSetConfig:   config.AzureConfig,
			Ignition:                   config.Ignition,
			InstallationName:           config.InstallationName,
			IPAMNetworkRange:           config.IPAMNetworkRange,
			ProjectName:                config.ProjectName,
			RegistryDomain:             config.RegistryDomain,
			OIDC:                       config.OIDC,
			SSOPublicKey:               config.SSOPublicKey,
			VMSSCheckWorkers:           config.VMSSCheckWorkers,
		}

		resourceSet, err = NewResourceSet(c)
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
