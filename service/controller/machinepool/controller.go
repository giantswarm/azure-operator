package machinepool

import (
	"context"
	"time"

	"github.com/giantswarm/certs/v4/pkg/certs"
	"github.com/giantswarm/conditions-handler/pkg/factory"
	conditionshandler "github.com/giantswarm/conditions-handler/pkg/handler"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/retryresource"
	"k8s.io/apimachinery/pkg/labels"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/pkg/label"
	"github.com/giantswarm/azure-operator/v8/pkg/project"
	"github.com/giantswarm/azure-operator/v8/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v8/service/controller/machinepool/handler/machinepooldependents"
	"github.com/giantswarm/azure-operator/v8/service/controller/machinepool/handler/machinepoolownerreference"
	"github.com/giantswarm/azure-operator/v8/service/controller/machinepool/handler/machinepoolupgrade"
	"github.com/giantswarm/azure-operator/v8/service/controller/machinepool/handler/nodestatus"
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
			Name: project.Name() + "-machine-pool-controller",
			NewRuntimeObjectFunc: func() ctrlClient.Object {
				return new(capiexp.MachinePool)
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

	var certsSearcher *certs.Searcher
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

	var cachedTenantClientFactory tenantcluster.Factory
	{
		tenantClientFactory, err := tenantcluster.NewFactory(certsSearcher, config.Logger)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		cachedTenantClientFactory, err = tenantcluster.NewCachedFactory(tenantClientFactory, config.Logger)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var machinePoolConditionsResource resource.Interface
	{
		c := conditionshandler.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
			Name:       "machinepoolconditions",
		}

		machinePoolConditionsResource, err = factory.NewMachinePoolConditionsHandler(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var machinepoolDependentsResource resource.Interface
	{
		c := machinepooldependents.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		machinepoolDependentsResource, err = machinepooldependents.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var ownerReferencesResource resource.Interface
	{
		c := machinepoolownerreference.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
			Scheme:     config.K8sClient.Scheme(),
		}

		ownerReferencesResource, err = machinepoolownerreference.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var nodestatusResource resource.Interface
	{
		c := nodestatus.Config{
			CtrlClient:          config.K8sClient.CtrlClient(),
			Logger:              config.Logger,
			TenantClientFactory: cachedTenantClientFactory,
		}

		nodestatusResource, err = nodestatus.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var machinepoolUpgradeResource resource.Interface
	{
		c := machinepoolupgrade.Config{
			CtrlClient:          config.K8sClient.CtrlClient(),
			Logger:              config.Logger,
			TenantClientFactory: cachedTenantClientFactory,
		}

		machinepoolUpgradeResource, err = machinepoolupgrade.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		machinePoolConditionsResource,
		machinepoolDependentsResource,
		ownerReferencesResource,
		nodestatusResource,
		machinepoolUpgradeResource,
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
