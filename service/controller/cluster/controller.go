package cluster

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
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller"
	"github.com/giantswarm/azure-operator/v5/service/controller/cluster/clusterconditions"
	"github.com/giantswarm/azure-operator/v5/service/controller/cluster/clusterdependents"
	"github.com/giantswarm/azure-operator/v5/service/controller/cluster/clusterownerreference"
	"github.com/giantswarm/azure-operator/v5/service/controller/cluster/clusterreleaseversion"
	"github.com/giantswarm/azure-operator/v5/service/controller/cluster/clusterupgrade"
	"github.com/giantswarm/azure-operator/v5/service/setting"
)

type ControllerConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger
	SentryDSN string

	Debug setting.Debug
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
		resources, err = NewClusterResourceSet(config)
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
			Name:      project.Name() + "-cluster-controller",
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(capiv1alpha3.Cluster)
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

func NewClusterResourceSet(config ControllerConfig) ([]resource.Interface, error) {
	var err error

	var clusterReleaseVersionResource resource.Interface
	{
		c := clusterreleaseversion.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		clusterReleaseVersionResource, err = clusterreleaseversion.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusterConditionsResource resource.Interface
	{
		c := clusterconditions.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		clusterConditionsResource, err = clusterconditions.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusterDependentsResource resource.Interface
	{
		c := clusterdependents.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		clusterDependentsResource, err = clusterdependents.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var ownerReferencesResource resource.Interface
	{
		c := clusterownerreference.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
			Scheme:     config.K8sClient.Scheme(),
		}

		ownerReferencesResource, err = clusterownerreference.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

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

	var clusterUpgradeResource resource.Interface
	{
		c := clusterupgrade.Config{
			CtrlClient:          config.K8sClient.CtrlClient(),
			Logger:              config.Logger,
			TenantClientFactory: tenantClientFactory,
		}

		clusterUpgradeResource, err = clusterupgrade.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		clusterReleaseVersionResource,
		clusterConditionsResource,
		clusterDependentsResource,
		ownerReferencesResource,
		clusterUpgradeResource,
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
