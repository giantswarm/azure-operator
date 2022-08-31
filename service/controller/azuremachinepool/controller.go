package azuremachinepool

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
	"k8s.io/apimachinery/pkg/labels"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v6/client"
	"github.com/giantswarm/azure-operator/v6/pkg/credential"
	"github.com/giantswarm/azure-operator/v6/pkg/employees"
	"github.com/giantswarm/azure-operator/v6/pkg/handler/ipam"
	"github.com/giantswarm/azure-operator/v6/pkg/handler/nodes"
	"github.com/giantswarm/azure-operator/v6/pkg/label"
	"github.com/giantswarm/azure-operator/v6/pkg/locker"
	"github.com/giantswarm/azure-operator/v6/pkg/project"
	"github.com/giantswarm/azure-operator/v6/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v6/service/collector"
	"github.com/giantswarm/azure-operator/v6/service/controller/azuremachinepool/handler/azuremachinepoolconditions"
	"github.com/giantswarm/azure-operator/v6/service/controller/azuremachinepool/handler/cloudconfigblob"
	"github.com/giantswarm/azure-operator/v6/service/controller/azuremachinepool/handler/migration"
	"github.com/giantswarm/azure-operator/v6/service/controller/azuremachinepool/handler/nodepool"
	"github.com/giantswarm/azure-operator/v6/service/controller/azuremachinepool/handler/spark"
	"github.com/giantswarm/azure-operator/v6/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v6/service/controller/internal/vmsku"
	"github.com/giantswarm/azure-operator/v6/service/controller/setting"
)

type ControllerConfig struct {
	APIServerSecurePort   int
	Azure                 setting.Azure
	AzureMetricsCollector collector.AzureAPIMetrics
	CalicoCIDRSize        int
	CalicoMTU             int
	CalicoSubnet          string
	ClusterIPRange        string
	CPAzureClientSet      *client.AzureClientSet
	CredentialProvider    credential.Provider
	DockerhubToken        string
	EtcdPrefix            string
	Ignition              setting.Ignition
	InstallationName      string
	K8sClient             k8sclient.Interface
	Locker                locker.Interface
	Logger                micrologger.Logger
	OIDC                  setting.OIDC
	RegistryDomain        string
	SentryDSN             string
	SSHUserList           employees.SSHUserList
	SSOPublicKey          string
	VMSSMSIEnabled        bool
}

func NewController(config ControllerConfig) (*controller.Controller, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.CPAzureClientSet == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.cpAzureClientSet must not be empty", config)
	}

	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}

	var err error

	var resources []resource.Interface
	{
		resources, err = NewAzureMachinePoolResourceSet(config)
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
			Name:      project.Name() + "-azure-machine-pool-controller",
			NewRuntimeObjectFunc: func() ctrlClient.Object {
				return new(capzexp.AzureMachinePool)
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

func NewAzureMachinePoolResourceSet(config ControllerConfig) ([]resource.Interface, error) {
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

	var azureMachinePoolConditionsResource resource.Interface
	{
		c := azuremachinepoolconditions.Config{
			AzureClientsFactory: &organizationClientFactory,
			CtrlClient:          config.K8sClient.CtrlClient(),
			Logger:              config.Logger,
		}

		azureMachinePoolConditionsResource, err = azuremachinepoolconditions.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var newDebugger *debugger.Debugger
	{
		c := debugger.Config{
			Logger: config.Logger,
		}

		newDebugger, err = debugger.New(c)
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

	nodesConfig := nodes.Config{
		CtrlClient: config.K8sClient.CtrlClient(),
		Debugger:   newDebugger,
		Logger:     config.Logger,

		Azure:         config.Azure,
		ClientFactory: organizationClientFactory,
	}

	var vmSKU *vmsku.VMSKUs
	{
		vmSKU, err = vmsku.New(vmsku.Config{
			AzureClientSet: config.CPAzureClientSet,
			Location:       config.Azure.Location,
			Logger:         config.Logger,
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var nodepoolResource resource.Interface
	{
		c := nodepool.Config{
			Config:              nodesConfig,
			CredentialProvider:  config.CredentialProvider,
			TenantClientFactory: cachedTenantClientFactory,
			VMSKU:               vmSKU,
		}

		nodepoolResource, err = nodepool.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var subnetChecker *ipam.AzureMachinePoolSubnetChecker
	{
		c := ipam.AzureMachinePoolSubnetCheckerConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		subnetChecker, err = ipam.NewAzureMachinePoolSubnetChecker(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var subnetPersister *ipam.AzureMachinePoolSubnetPersister
	{
		c := ipam.AzureMachinePoolSubnetPersisterConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		subnetPersister, err = ipam.NewAzureMachinePoolSubnetPersister(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var subnetReleaser *ipam.AzureMachinePoolSubnetReleaser
	{
		c := ipam.AzureMachinePoolSubnetReleaserConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		subnetReleaser, err = ipam.NewAzureMachinePoolSubnetReleaser(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var subnetCollector *ipam.AzureMachinePoolSubnetCollector
	{
		c := ipam.AzureMachinePoolSubnetCollectorConfig{
			AzureClientFactory: organizationClientFactory,
			CtrlClient:         config.K8sClient.CtrlClient(),
			Logger:             config.Logger,
		}

		subnetCollector, err = ipam.NewAzureMachineSubnetCollector(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var networkRangeGetter *ipam.AzureMachinePoolNetworkRangeGetter
	{
		c := ipam.AzureMachinePoolNetworkRangeGetterConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		networkRangeGetter, err = ipam.NewAzureMachinePoolNetworkRangeGetter(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var ipamResource resource.Interface
	{
		c := ipam.Config{
			Checker:            subnetChecker,
			Collector:          subnetCollector,
			Locker:             config.Locker,
			Logger:             config.Logger,
			NetworkRangeGetter: networkRangeGetter,
			NetworkRangeType:   ipam.SubnetRange,
			Persister:          subnetPersister,
			Releaser:           subnetReleaser,
		}

		ipamResource, err = ipam.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var sparkResource resource.Interface
	{
		c := spark.Config{
			APIServerSecurePort: config.APIServerSecurePort,
			Azure:               config.Azure,
			CalicoCIDRSize:      config.CalicoCIDRSize,
			CalicoMTU:           config.CalicoMTU,
			CalicoSubnet:        config.CalicoSubnet,
			CertsSearcher:       certsSearcher,
			ClientFactory:       organizationClientFactory,
			ClusterIPRange:      config.ClusterIPRange,
			CredentialProvider:  config.CredentialProvider,
			CtrlClient:          config.K8sClient.CtrlClient(),
			DockerhubToken:      config.DockerhubToken,
			EtcdPrefix:          config.EtcdPrefix,
			Ignition:            config.Ignition,
			Logger:              config.Logger,
			OIDC:                config.OIDC,
			RegistryDomain:      config.RegistryDomain,
			SSHUserList:         config.SSHUserList,
			SSOPublicKey:        config.SSOPublicKey,
		}

		sparkResource, err = spark.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var cloudconfigblobResource resource.Interface
	{
		c := cloudconfigblob.Config{
			ClientFactory: organizationClientFactory,
			CtrlClient:    config.K8sClient.CtrlClient(),
			Logger:        config.Logger,
		}

		cloudconfigblobResource, err = cloudconfigblob.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		migrationResource,
		azureMachinePoolConditionsResource,
		ipamResource,
		sparkResource,
		cloudconfigblobResource,
		nodepoolResource,
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
