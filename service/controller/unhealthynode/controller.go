package unhealthynode

import (
	"context"
	"time"

	"github.com/giantswarm/certs/v4/pkg/certs"
	"github.com/giantswarm/k8sclient/v7/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller"
	"github.com/giantswarm/operatorkit/v7/pkg/resource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/wrapper/retryresource"
	"github.com/giantswarm/tenantcluster/v6/pkg/tenantcluster"
	"k8s.io/apimachinery/pkg/labels"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/client"
	"github.com/giantswarm/azure-operator/v8/pkg/credential"
	"github.com/giantswarm/azure-operator/v8/pkg/label"
	"github.com/giantswarm/azure-operator/v8/pkg/project"
	"github.com/giantswarm/azure-operator/v8/service/collector"
	"github.com/giantswarm/azure-operator/v8/service/controller/unhealthynode/handler/terminateunhealthynode"
)

type ControllerConfig struct {
	K8sClient k8sclient.Interface
	Logger    micrologger.Logger

	AzureMetricsCollector collector.AzureAPIMetrics
	CredentialProvider    credential.Provider
	SentryDSN             string
}

type Controller struct {
	*controller.Controller
}

func NewController(config ControllerConfig) (*controller.Controller, error) {
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

	var certsSearcher *certs.Searcher
	{
		c := certs.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			WatchTimeout: 30 * time.Second,
		}

		certsSearcher, err = certs.NewSearcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resources []resource.Interface
	{
		resources, err = newTerminateUnhealthyNodeResources(config, certsSearcher)
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
			Name: project.Name() + "-terminate-unhealthy-node-controller",
			NewRuntimeObjectFunc: func() ctrlClient.Object {
				return new(capi.Cluster)
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

func newTerminateUnhealthyNodeResources(config ControllerConfig, certsSearcher *certs.Searcher) ([]resource.Interface, error) {
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

	var tenantRestConfigProvider *tenantcluster.TenantCluster
	{
		c := tenantcluster.Config{
			CertID:        certs.APICert,
			CertsSearcher: certsSearcher,
			Logger:        config.Logger,
		}

		tenantRestConfigProvider, err = tenantcluster.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var terminateUnhealthyNodeResource resource.Interface
	{
		c := terminateunhealthynode.Config{
			AzureClientsFactory:      &organizationClientFactory,
			CtrlClient:               config.K8sClient.CtrlClient(),
			Logger:                   config.Logger,
			TenantRestConfigProvider: tenantRestConfigProvider,
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
