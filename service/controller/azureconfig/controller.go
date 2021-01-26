package azureconfig

import (
	"context"
	"net"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs/v3/pkg/certs"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v4/pkg/controller"
	"github.com/giantswarm/operatorkit/v4/pkg/resource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/crud"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/wrapper/retryresource"
	"github.com/giantswarm/tenantcluster/v3/pkg/tenantcluster"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/pkg/credential"
	"github.com/giantswarm/azure-operator/v5/pkg/employees"
	"github.com/giantswarm/azure-operator/v5/pkg/handler/ipam"
	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes"
	"github.com/giantswarm/azure-operator/v5/pkg/handler/release"
	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/locker"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/collector"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/azureconfigfinalizer"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/capzcrs"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/clusterid"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/masters"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/namespace"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/resourcegroup"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/service"
	"github.com/giantswarm/azure-operator/v5/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/blobobject"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/containerurl"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/deployment"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/dnsrecord"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/encryptionkey"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/endpoints"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/vpn"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/vpnconnection"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/workermigration"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"
)

type ControllerConfig struct {
	CredentialProvider credential.Provider
	InstallationName   string
	K8sClient          k8sclient.Interface
	Locker             locker.Interface
	Logger             micrologger.Logger

	Azure                 setting.Azure
	AzureMetricsCollector collector.AzureAPIMetrics
	// Azure client set used when managing control plane resources
	CPAzureClientSet *client.AzureClientSet
	ProjectName      string

	ClusterVNetMaskBits int

	Ignition          setting.Ignition
	IPAMNetworkRange  net.IPNet
	IPAMReservedCIDRs []net.IPNet
	OIDC              setting.OIDC
	SSHUserList       employees.SSHUserList
	SSOPublicKey      string
	TemplateVersion   string

	DockerhubToken  string
	RegistryDomain  string
	RegistryMirrors []string

	Debug     setting.Debug
	SentryDSN string
}

type Controller struct {
	*controller.Controller
}

func NewController(config ControllerConfig) (*controller.Controller, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

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
		resources, err = newAzureConfigResources(config, certsSearcher)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var operatorkitController *controller.Controller
	{
		c := controller.Config{
			InitCtx: func(ctx context.Context, obj interface{}) (context.Context, error) {
				cr, err := key.ToCustomResource(obj)
				if err != nil {
					return nil, microerror.Mask(err)
				}

				organizationAzureClientCredentialsConfig, subscriptionID, partnerID, err := config.CredentialProvider.GetOrganizationAzureCredentials(ctx, key.CredentialNamespace(cr), key.CredentialName(cr))
				if err != nil {
					return nil, microerror.Mask(err)
				}

				tenantClusterAzureClientSet, err := client.NewAzureClientSet(organizationAzureClientCredentialsConfig, config.AzureMetricsCollector, subscriptionID, partnerID)
				if err != nil {
					return nil, microerror.Mask(err)
				}

				var cloudConfig *cloudconfig.CloudConfig
				{
					c := cloudconfig.Config{
						Azure:                  config.Azure,
						AzureClientCredentials: organizationAzureClientCredentialsConfig,
						CtrlClient:             config.K8sClient.CtrlClient(),
						DockerhubToken:         config.DockerhubToken,
						Ignition:               config.Ignition,
						Logger:                 config.Logger,
						OIDC:                   config.OIDC,
						RegistryMirrors:        config.RegistryMirrors,
						SSOPublicKey:           config.SSOPublicKey,
						SubscriptionID:         subscriptionID,
					}

					cloudConfig, err = cloudconfig.New(c)
					if err != nil {
						return nil, microerror.Mask(err)
					}
				}

				c := controllercontext.Context{
					AzureClientSet: tenantClusterAzureClientSet,
					CloudConfig:    cloudConfig,
				}
				ctx = controllercontext.NewContext(ctx, c)

				return ctx, nil
			},
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
			Name:      project.Name() + "-azureconfig-controller",
			NewRuntimeObjectFunc: func() runtime.Object {
				return new(v1alpha1.AzureConfig)
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

func newAzureConfigResources(config ControllerConfig, certsSearcher certs.Interface) ([]resource.Interface, error) {
	var err error

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

	var azureconfigFinalizerResource resource.Interface
	{
		c := azureconfigfinalizer.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		azureconfigFinalizerResource, err = azureconfigfinalizer.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var clusteridResource resource.Interface
	{
		c := clusterid.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		clusteridResource, err = clusterid.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var capzcrsResource resource.Interface
	{
		c := capzcrs.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,

			Location: config.Azure.Location,
		}

		capzcrsResource, err = capzcrs.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var releaseResource resource.Interface
	{
		c := release.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		releaseResource, err = release.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var resourceGroupResource resource.Interface
	{
		c := resourcegroup.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,

			Azure:            config.Azure,
			InstallationName: config.InstallationName,
		}

		resourceGroupResource, err = resourcegroup.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var containerURLResource resource.Interface
	{
		c := containerurl.Config{
			Logger: config.Logger,
		}

		containerURLResource, err = containerurl.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var encryptionkeyResource resource.Interface
	{
		c := encryptionkey.Config{
			K8sClient:   config.K8sClient.K8sClient(),
			Logger:      config.Logger,
			ProjectName: config.ProjectName,
		}

		encryptionkeyResource, err = encryptionkey.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var blobObjectResource resource.Interface
	{
		c := blobobject.Config{
			CertsSearcher:  certsSearcher,
			CtrlClient:     config.K8sClient.CtrlClient(),
			G8sClient:      config.K8sClient.G8sClient(),
			K8sClient:      config.K8sClient.K8sClient(),
			Logger:         config.Logger,
			RegistryDomain: config.RegistryDomain,
			SSHUserList:    config.SSHUserList,
		}

		blobObject, err := blobobject.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		blobObjectResource, err = toCRUDResource(config.Logger, blobObject)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var deploymentResource resource.Interface
	{
		c := deployment.Config{
			Debugger:         newDebugger,
			G8sClient:        config.K8sClient.G8sClient(),
			InstallationName: config.InstallationName,
			Logger:           config.Logger,

			Azure:                      config.Azure,
			AzureClientSet:             config.CPAzureClientSet,
			ClientFactory:              organizationClientFactory,
			ControlPlaneSubscriptionID: config.CPAzureClientSet.SubscriptionID,
			Debug:                      config.Debug,
		}

		deploymentResource, err = deployment.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var dnsrecordResource resource.Interface
	{
		c := dnsrecord.Config{
			CPRecordSetsClient: *config.CPAzureClientSet.DNSRecordSetsClient,
			Logger:             config.Logger,
		}

		ops, err := dnsrecord.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		dnsrecordResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var endpointsResource resource.Interface
	{
		c := endpoints.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		ops, err := endpoints.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		endpointsResource, err = toCRUDResource(config.Logger, ops)
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

	var mastersResource resource.Interface
	{
		c := masters.Config{
			Config:                   nodesConfig,
			CtrlClient:               config.K8sClient.CtrlClient(),
			TenantRestConfigProvider: tenantRestConfigProvider,
		}

		mastersResource, err = masters.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var workerMigrationResource resource.Interface
	{
		c := workermigration.Config{
			CertsSearcher:             certsSearcher,
			ClientFactory:             clientFactory,
			CPPublicIPAddressesClient: config.CPAzureClientSet.PublicIpAddressesClient,
			CtrlClient:                config.K8sClient.CtrlClient(),
			Logger:                    config.Logger,

			InstallationName: config.InstallationName,
			Location:         config.Azure.Location,
		}

		workerMigrationResource, err = workermigration.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var azureConfigChecker *ipam.AzureConfigChecker
	{
		c := ipam.AzureConfigCheckerConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		azureConfigChecker, err = ipam.NewAzureConfigChecker(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var azureConfigPersister *ipam.AzureConfigPersister
	{
		c := ipam.AzureConfigPersisterConfig{
			CtrlClient: config.K8sClient.CtrlClient(),
			Logger:     config.Logger,
		}

		azureConfigPersister, err = ipam.NewAzureConfigPersister(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var virtualNetworkCollector *ipam.VirtualNetworkCollector
	{
		c := ipam.VirtualNetworkCollectorConfig{
			AzureMetricsCollector: config.AzureMetricsCollector,
			CredentialProvider:    config.CredentialProvider,
			K8sClient:             config.K8sClient,
			InstallationName:      config.InstallationName,
			Logger:                config.Logger,

			NetworkRange:  config.IPAMNetworkRange,
			ReservedCIDRs: config.IPAMReservedCIDRs,
		}

		virtualNetworkCollector, err = ipam.NewVirtualNetworkCollector(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var networkRangeGetter *ipam.AzureConfigNetworkRangeGetter
	{
		c := ipam.AzureConfigNetworkRangeGetterConfig{
			InstallationNetworkRange:            config.IPAMNetworkRange,
			TenantClusterVirtualNetworkMaskBits: config.ClusterVNetMaskBits,
		}

		networkRangeGetter, err = ipam.NewAzureConfigNetworkRangeGetter(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var ipamResource resource.Interface
	{
		c := ipam.Config{
			Checker:            azureConfigChecker,
			Collector:          virtualNetworkCollector,
			Locker:             config.Locker,
			Logger:             config.Logger,
			NetworkRangeGetter: networkRangeGetter,
			NetworkRangeType:   ipam.VirtualNetworkRange,
			Persister:          azureConfigPersister,
			Releaser:           ipam.NewNOPReleaser(),
		}

		ipamResource, err = ipam.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceResource resource.Interface
	{
		c := namespace.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		ops, err := namespace.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		namespaceResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var serviceResource resource.Interface
	{
		c := service.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		ops, err := service.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		serviceResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var vpnResource resource.Interface
	{
		c := vpn.Config{
			CtrlClient: config.K8sClient.CtrlClient(),
			Debugger:   newDebugger,
			Logger:     config.Logger,

			Azure: config.Azure,
		}

		vpnResource, err = vpn.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var vpnconnectionResource resource.Interface
	{
		c := vpnconnection.Config{
			Azure:                                    config.Azure,
			Logger:                                   config.Logger,
			CPVirtualNetworkGatewaysClient:           *config.CPAzureClientSet.VirtualNetworkGatewaysClient,
			CPVirtualNetworkGatewayConnectionsClient: *config.CPAzureClientSet.VirtualNetworkGatewayConnectionsClient,
		}

		ops, err := vpnconnection.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		vpnconnectionResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []resource.Interface{
		azureconfigFinalizerResource,
		clusteridResource,
		capzcrsResource,
		namespaceResource,
		ipamResource,
		releaseResource,
		serviceResource,
		resourceGroupResource,
		encryptionkeyResource,
		deploymentResource,
		containerURLResource,
		blobObjectResource,
		dnsrecordResource,
		mastersResource,
		workerMigrationResource,
		endpointsResource,
		vpnResource,
		vpnconnectionResource,
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

func toCRUDResource(logger micrologger.Logger, v crud.Interface) (*crud.Resource, error) {
	c := crud.ResourceConfig{
		CRUD:   v,
		Logger: logger,
	}

	r, err := crud.NewResource(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r, nil
}
