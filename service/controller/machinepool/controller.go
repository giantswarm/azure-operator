package machinepool

import (
	"context"
	"time"

	"github.com/giantswarm/certs/v3/pkg/certs"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	operatorkitcontroller "github.com/giantswarm/operatorkit/v4/pkg/controller"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/retryresource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller"
	"github.com/giantswarm/azure-operator/v5/service/controller/machinepool/machinepoolconditions"
	"github.com/giantswarm/azure-operator/v5/service/controller/machinepool/machinepooldependents"
	"github.com/giantswarm/azure-operator/v5/service/controller/machinepool/machinepoolownerreference"
	"github.com/giantswarm/azure-operator/v5/service/controller/machinepool/machinepoolupgrade"
	"github.com/giantswarm/azure-operator/v5/service/controller/machinepool/nodestatus"
)

type ControllerConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
	SentryDSN string
}

func NewController(config ControllerConfig) (*operatorkitcontroller.Controller, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(controller.InvalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(controller.InvalidConfigError, "%T.K8sClient must not be empty", config)
	}

	var err error

	var resources []resource.Interface
	{
		resources, err = NewMachinePoolResourceSet(config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *operatorkitcontroller.Controller
	{
		c := operatorkitcontroller.Config{
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

		operatorkitController, err = operatorkitcontroller.New(c)
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

	var tenantClientFactory tenantcluster.Factory
	{
		tenantClientFactory, err = tenantcluster.NewFactory(certsSearcher, config.Logger)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var machinePoolConditionsResource resource.Interface
	{
		c := machinepoolconditions.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		machinePoolConditionsResource, err = machinepoolconditions.New(c)
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
			TenantClientFactory: tenantClientFactory,
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
			TenantClientFactory: tenantClientFactory,
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
