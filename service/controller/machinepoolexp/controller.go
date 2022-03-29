package machinepoolexp

import (
	"context"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/retryresource"
	"k8s.io/apimachinery/pkg/labels"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/controller/machinepoolexp/handler/migration"
)

type ControllerConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
	SentryDSN string
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
		resources, err = NewMachinePoolResourceSet(config)
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
			Name: project.Name() + "-machine-pool-exp-controller",
			NewRuntimeObjectFunc: func() ctrlClient.Object {
				return new(oldcapiexpv1alpha3.MachinePool)
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

func NewMachinePoolResourceSet(config ControllerConfig) ([]resource.Interface, error) {
	var err error
	var migrationResource resource.Interface
	{
		c := migration.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}
		migrationResource, err = migration.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		migrationResource,
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
