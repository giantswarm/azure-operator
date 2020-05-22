package controller

import (
	"context"
	"fmt"

	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"github.com/spf13/viper"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/flag"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/azureconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type AzureClusterResourceSetConfig struct {
	CertsSearcher certs.Interface
	K8sClient     k8sclient.Interface
	Logger        micrologger.Logger

	Flag  *flag.Flag
	Viper *viper.Viper

	Azure            setting.Azure
	CPAzureClientSet client.AzureClientSet
	Ignition         setting.Ignition
	InstallationName string
	ProjectName      string
	RegistryDomain   string
	OIDC             setting.OIDC
	SSOPublicKey     string
	VMSSCheckWorkers int
}

func NewAzureClusterResourceSet(config AzureClusterResourceSetConfig) (*controller.ResourceSet, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	/*
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
	*/

	var crMapperResource *azureconfig.Resource
	{
		c := azureconfig.Config{
			Logger: config.Logger,

			Flag:  config.Flag,
			Viper: config.Viper,

			CtrlClient: config.K8sClient.CtrlClient(),
		}

		crMapperResource, err = azureconfig.New(c)
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
		cr, err := key.ToAzureCluster(obj)
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
		return context.Background(), nil
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
