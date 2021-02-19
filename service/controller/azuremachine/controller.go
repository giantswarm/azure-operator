package azuremachine

import (
	"context"

	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/controller"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsawarefactory"
	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/collector"
	"github.com/giantswarm/azure-operator/v5/service/controller/azuremachine/handler/azuremachineconditions"
	"github.com/giantswarm/azure-operator/v5/service/controller/azuremachine/handler/azuremachinemetadata"
)

type ControllerConfig struct {
	AzureMetricsCollector collector.AzureAPIMetrics
	MCAzureClientFactory  credentialsawarefactory.Interface
	WCAzureClientFactory  credentialsawarefactory.Interface
	K8sClient             k8sclient.Interface
	Logger                micrologger.Logger
	SentryDSN             string
}

func NewController(config ControllerConfig) (*controller.Controller, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.MCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}
	if config.WCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}

	var err error

	var resources []resource.Interface
	{
		resources, err = NewAzureMachineResourceSet(config)
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
			Name:      project.Name() + "-azure-machine-controller",
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(capz.AzureMachine)
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

func NewAzureMachineResourceSet(config ControllerConfig) ([]resource.Interface, error) {
	var err error

	var azureMachineMetadataResource resource.Interface
	{
		c := azuremachinemetadata.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		azureMachineMetadataResource, err = azuremachinemetadata.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}
	var azureMachineConditionsResource resource.Interface
	{
		c := azuremachineconditions.Config{
			WCAzureClientsFactory: config.WCAzureClientFactory,
			CtrlClient:            config.K8sClient.CtrlClient(),
			Logger:                config.Logger,
		}

		azureMachineConditionsResource, err = azuremachineconditions.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		azureMachineMetadataResource,
		azureMachineConditionsResource,
	}

	return resources, nil
}
