package controller

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"github.com/giantswarm/randomkeys"
	"github.com/spf13/viper"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/flag"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/crmapper"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
	"github.com/giantswarm/azure-operator/v4/service/credential"
	"github.com/giantswarm/azure-operator/v4/service/network"
)

type AzureClusterResourceSetConfig struct {
	CertsSearcher certs.Interface
	K8sClient     k8sclient.Interface
	Logger        micrologger.Logger

	Flag  *flag.Flag
	Viper *viper.Viper

	Azure                    setting.Azure
	HostAzureClientSetConfig client.AzureClientSetConfig
	Ignition                 setting.Ignition
	InstallationName         string
	ProjectName              string
	RegistryDomain           string
	OIDC                     setting.OIDC
	SSOPublicKey             string
	VMSSCheckWorkers         int
}

func NewAzureClusterResourceSet(config AzureClusterResourceSetConfig) (*controller.ResourceSet, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	var certsSearcher certs.Interface
	{
		c := certs.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			WatchTimeout: 5 * time.Second,
		}

		certsSearcher, err = certs.NewSearcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var randomkeysSearcher *randomkeys.Searcher
	{
		c := randomkeys.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		randomkeysSearcher, err = randomkeys.NewSearcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var crMapperResource *crmapper.Resource
	{
		c := crmapper.Config{
			Logger: config.Logger,

			Flag:  config.Flag,
			Viper: config.Viper,

			CtrlClient: config.K8sClient.CtrlClient(),
		}

		crMapperResource, err = crmapper.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		crMapperResource,
	}

	{
		c := retryresource.WrapConfig{
			Logger: config.Logger,
		}

		resources, err = retryresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		c := metricsresource.WrapConfig{}
		resources, err = metricsresource.Wrap(resources, c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	handlesFunc := func(obj interface{}) bool {
		cr, err := key.ToCustomResource(obj)
		if err != nil {
			config.Logger.Log("level", "warning", "message", fmt.Sprintf("invalid object: %s", err), "stack", fmt.Sprintf("%v", err)) // nolint: errcheck
			return false
		}

		if key.OperatorVersion(&cr) == project.Version() {
			return true
		}

		return false
	}

	initCtxFunc := func(ctx context.Context, obj interface{}) (context.Context, error) {
		cr, err := key.ToCustomResource(obj)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		_, vnet, err := net.ParseCIDR(key.VnetCIDR(cr))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		subnets, err := network.Compute(*vnet)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		guestAzureClientSetConfig, err := credential.GetAzureConfig(config.K8sClient.K8sClient(), key.CredentialName(cr), key.CredentialNamespace(cr))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		guestAzureClientSetConfig.EnvironmentName = config.Azure.EnvironmentName

		azureClients, err := client.NewAzureClientSet(*guestAzureClientSetConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		var cloudConfig *cloudconfig.CloudConfig
		{
			c := cloudconfig.Config{
				CertsSearcher:      certsSearcher,
				Logger:             config.Logger,
				RandomkeysSearcher: randomkeysSearcher,

				Azure:        config.Azure,
				AzureConfig:  *guestAzureClientSetConfig,
				AzureNetwork: *subnets,
				Ignition:     config.Ignition,
				OIDC:         config.OIDC,
				SSOPublicKey: config.SSOPublicKey,
			}

			cloudConfig, err = cloudconfig.New(c)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}

		c := controllercontext.Context{
			AzureClientSet: azureClients,
			AzureNetwork:   subnets,
			CloudConfig:    cloudConfig,
		}
		ctx = controllercontext.NewContext(ctx, c)

		return ctx, nil
	}

	var resourceSet *controller.ResourceSet
	{
		c := controller.ResourceSetConfig{
			Handles:   handlesFunc,
			InitCtx:   initCtxFunc,
			Logger:    config.Logger,
			Resources: resources,
		}

		resourceSet, err = controller.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceSet, nil
}
