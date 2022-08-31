package azuremachine

import (
	"context"
	"time"

	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	"k8s.io/apimachinery/pkg/labels"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v6/client"
	"github.com/giantswarm/azure-operator/v6/pkg/credential"
	"github.com/giantswarm/azure-operator/v6/pkg/label"
	"github.com/giantswarm/azure-operator/v6/pkg/project"
	"github.com/giantswarm/azure-operator/v6/service/collector"
	"github.com/giantswarm/azure-operator/v6/service/controller/azuremachine/handler/azuremachineconditions"
	"github.com/giantswarm/azure-operator/v6/service/controller/azuremachine/handler/azuremachinemetadata"
)

type ControllerConfig struct {
	AzureMetricsCollector collector.AzureAPIMetrics
	CredentialProvider    credential.Provider
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
			NewRuntimeObjectFunc: func() ctrlClient.Object {
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
			AzureClientsFactory: &organizationClientFactory,
			CtrlClient:          config.K8sClient.CtrlClient(),
			Logger:              config.Logger,
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
