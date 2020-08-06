package controller

import (
	"context"
	"net"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/resource"
	"github.com/giantswarm/operatorkit/resource/crud"
	"github.com/giantswarm/operatorkit/resource/wrapper/metricsresource"
	"github.com/giantswarm/operatorkit/resource/wrapper/retryresource"
	"github.com/giantswarm/randomkeys"
	"github.com/giantswarm/statusresource"
	"github.com/giantswarm/tenantcluster"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/locker"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/blobobject"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/clusterid"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/containerurl"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/deployment"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/dnsrecord"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/encryptionkey"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/endpoints"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/instance"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/ipam"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/masters"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/namespace"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/release"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/resourcegroup"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/service"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/tenantclients"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/vpn"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/vpnconnection"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

type AzureConfigConfig struct {
	CredentialProvider credential.Provider
	InstallationName   string
	K8sClient          k8sclient.Interface
	Locker             locker.Interface
	Logger             micrologger.Logger

	Azure setting.Azure
	// Azure client set used when managing control plane resources
	CPAzureClientSet *client.AzureClientSet
	// Azure credentials used to create Azure client set for tenant clusters
	GSClientCredentialsConfig auth.ClientCredentialsConfig
	ProjectName               string
	RegistryDomain            string
	RegistryMirrors           []string

	GuestSubnetMaskBits int

	Ignition         setting.Ignition
	IPAMNetworkRange net.IPNet
	OIDC             setting.OIDC
	SSOPublicKey     string
	TemplateVersion  string
	VMSSCheckWorkers int

	Debug     setting.Debug
	SentryDSN string
}

type AzureConfig struct {
	*controller.Controller
}

func NewAzureConfig(config AzureConfigConfig) (*controller.Controller, error) {
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

			WatchTimeout: 5 * time.Second,
		}

		certsSearcher, err = certs.NewSearcher(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var randomkeysSearcher *randomkeys.Searcher
	{
		c := randomkeys.Config{
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,
		}

		randomkeysSearcher, err = randomkeys.NewSearcher(c)
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

				tenantClusterAzureClientSet, err := client.NewAzureClientSet(organizationAzureClientCredentialsConfig, subscriptionID, partnerID)
				if err != nil {
					return nil, microerror.Mask(err)
				}

				var cloudConfig *cloudconfig.CloudConfig
				{
					c := cloudconfig.Config{
						CertsSearcher:      certsSearcher,
						Logger:             config.Logger,
						RandomkeysSearcher: randomkeysSearcher,

						Azure:                  config.Azure,
						AzureClientCredentials: organizationAzureClientCredentialsConfig,
						Ignition:               config.Ignition,
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

func newAzureConfigResources(config AzureConfigConfig, certsSearcher certs.Interface) ([]resource.Interface, error) {
	var err error

	var clientFactory *client.Factory
	{
		c := client.FactoryConfig{
			CacheDuration:      30 * time.Minute,
			CredentialProvider: config.CredentialProvider,
			Logger:             config.Logger,
		}

		clientFactory, err = client.NewFactory(c)
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

	var tenantCluster tenantcluster.Interface
	{
		c := tenantcluster.Config{
			CertsSearcher: certsSearcher,
			Logger:        config.Logger,

			CertID: certs.APICert,
		}

		tenantCluster, err = tenantcluster.New(c)
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

	var statusResource resource.Interface
	{
		c := statusresource.ResourceConfig{
			ClusterEndpointFunc:      key.ToClusterEndpoint,
			ClusterIDFunc:            key.ToClusterID,
			ClusterStatusFunc:        key.ToClusterStatus,
			NodeCountFunc:            key.ToNodeCount,
			Logger:                   config.Logger,
			RESTClient:               config.K8sClient.G8sClient().ProviderV1alpha1().RESTClient(),
			TenantCluster:            tenantCluster,
			VersionBundleVersionFunc: key.ToOperatorVersion,
		}

		statusResource, err = statusresource.NewResource(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var tenantClientsResource resource.Interface
	{
		c := tenantclients.Config{
			Logger: config.Logger,
			Tenant: tenantCluster,
		}

		tenantClientsResource, err = tenantclients.New(c)
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
			Logger: config.Logger,

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
			G8sClient:      config.K8sClient.G8sClient(),
			K8sClient:      config.K8sClient.K8sClient(),
			Logger:         config.Logger,
			RegistryDomain: config.RegistryDomain,
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
			ClientFactory:              clientFactory,
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

	var iwd vmsscheck.InstanceWatchdog
	{
		c := vmsscheck.Config{
			Logger:     config.Logger,
			NumWorkers: config.VMSSCheckWorkers,
		}

		var err error
		iwd, err = vmsscheck.NewInstanceWatchdog(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	nodesConfig := nodes.Config{
		Debugger:  newDebugger,
		G8sClient: config.K8sClient.G8sClient(),
		K8sClient: config.K8sClient.K8sClient(),
		Logger:    config.Logger,

		Azure:            config.Azure,
		ClientFactory:    clientFactory,
		InstanceWatchdog: iwd,
	}

	var mastersResource resource.Interface
	{
		c := masters.Config{
			Config: nodesConfig,
		}

		mastersResource, err = masters.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var instanceResource resource.Interface
	{
		c := instance.Config{
			Config: nodesConfig,
		}

		instanceResource, err = instance.New(c)
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

	var subnetCollector *ipam.SubnetCollector
	{
		c := ipam.SubnetCollectorConfig{
			CredentialProvider: config.CredentialProvider,
			K8sClient:          config.K8sClient,
			InstallationName:   config.InstallationName,
			Logger:             config.Logger,

			NetworkRange: config.IPAMNetworkRange,
		}

		subnetCollector, err = ipam.NewSubnetCollector(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var ipamResource resource.Interface
	{
		c := ipam.Config{
			Checker:   azureConfigChecker,
			Collector: subnetCollector,
			Locker:    config.Locker,
			Logger:    config.Logger,
			Persister: azureConfigPersister,

			AllocatedSubnetMaskBits: config.GuestSubnetMaskBits,
			NetworkRange:            config.IPAMNetworkRange,
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
			Debugger: newDebugger,
			Logger:   config.Logger,

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
		clusteridResource,
		namespaceResource,
		ipamResource,
		statusResource,
		releaseResource,
		tenantClientsResource,
		serviceResource,
		resourceGroupResource,
		encryptionkeyResource,
		deploymentResource,
		containerURLResource,
		blobObjectResource,
		dnsrecordResource,
		mastersResource,
		instanceResource,
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
