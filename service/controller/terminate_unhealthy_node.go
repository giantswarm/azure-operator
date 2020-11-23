package controller

import (
	"context"
	"time"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/controller"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/retryresource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/pkg/credential"
	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/collector"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/terminateunhealthynode"
)

type TerminateUnhealthyNodeConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	AzureMetricsCollector collector.AzureAPIMetrics
	CredentialProvider    credential.Provider
	SentryDSN             string
}

type TerminateUnhealthyNode struct {
	*controller.Controller
}

func NewTerminateUnhealthyNode(config TerminateUnhealthyNodeConfig) (*controller.Controller, error) {
	var err error

	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.AzureMetricsCollector == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureMetricsCollector must not be empty", config)
	}
	if config.CredentialProvider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CredentialProvider must not be empty", config)
	}

	var resources []resource.Interface
	{
		resources, err = newTerminateUnhealthyNodeResources(config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			InitCtx: func(ctx context.Context, obj interface{}) (context.Context, error) {
				return ctx, nil
			},
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			// Name is used to compute finalizer names. This results in something
			// like operatorkit.giantswarm.io/azure-operator-machine-pool-controller.
			Name: project.Name() + "-machine-pool-controller",
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(expcapiv1alpha3.MachinePool)
			},
			Resources: resources,
			Selector: labels.SelectorFromSet(map[string]string{
				label.OperatorVersion: project.Version(),
			}),
			SentryDSN: config.SentryDSN,
		}

		operatorkitController, err = controller.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return operatorkitController, nil
}

func newTerminateUnhealthyNodeResources(config TerminateUnhealthyNodeConfig) ([]resource.Interface, error) {
	var err error

	var clientFactory *client.Factory
	{
		c := client.FactoryConfig{
			AzureAPIMetrics:    config.AzureMetricsCollector,
			CacheDuration:      30 * time.Minute,
			CredentialProvider: config.CredentialProvider,
			Logger:             config.Logger,
		}

		clientFactory, err = client.NewFactory(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var organizationClientFactory client.OrganizationFactory
	{
		c := client.OrganizationFactoryConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Factory:    clientFactory,
			Logger:     config.Logger,
		}
		organizationClientFactory = client.NewOrganizationFactory(c)
	}

	var terminateUnhealthyNodeResource resource.Interface
	{
		c := terminateunhealthynode.Config{
			AzureClientsFactory: &organizationClientFactory,
			CtrlClient:          config.K8sClient.CtrlClient(),
			Logger:              config.Logger,
		}

		terminateUnhealthyNodeResource, err = terminateunhealthynode.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		terminateUnhealthyNodeResource,
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

	return resources, nil
}