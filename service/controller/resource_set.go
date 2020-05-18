package controller

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/giantswarm/certs"
	"github.com/giantswarm/k8sclient"
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

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/cloudconfig"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/blobobject"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/containerurl"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/deployment"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/dnsrecord"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/encryptionkey"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/endpoints"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/instance"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/masters"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/namespace"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/release"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/resourcegroup"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/service"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/tenantclients"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/vpn"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/vpnconnection"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
	"github.com/giantswarm/azure-operator/v4/service/credential"
	"github.com/giantswarm/azure-operator/v4/service/network"
)

type ResourceSetConfig struct {
	CertsSearcher certs.Interface
	K8sClient     k8sclient.Interface
	Logger        micrologger.Logger

	Azure                    setting.Azure
	HostAzureClientSetConfig client.AzureClientSetConfig
	Ignition                 setting.Ignition
	InstallationName         string
	ProjectName              string
	RegistryDomain           string
	OIDC                     setting.OIDC
	SSOPublicKey             string
	VMSSCheckWorkers         int
}

func NewResourceSet(config ResourceSetConfig) (*controller.ResourceSet, error) {
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	var certsSearcher certs.Interface
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
			CertsSearcher:  config.CertsSearcher,
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
			Debugger:  newDebugger,
			G8sClient: config.K8sClient.G8sClient(),
			Logger:    config.Logger,

			Azure: config.Azure,
		}

		deploymentResource, err = deployment.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var dnsrecordResource resource.Interface
	{
		c := dnsrecord.Config{
			Logger: config.Logger,

			HostAzureClientSetConfig: config.HostAzureClientSetConfig,
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

	var mastersResource resource.Interface
	{
		c := masters.Config{
			Debugger:  newDebugger,
			G8sClient: config.K8sClient.G8sClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			Azure:            config.Azure,
			InstanceWatchdog: iwd,
		}

		mastersResource, err = masters.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var instanceResource resource.Interface
	{
		c := instance.Config{
			Debugger:  newDebugger,
			G8sClient: config.K8sClient.G8sClient(),
			K8sClient: config.K8sClient.K8sClient(),
			Logger:    config.Logger,

			Azure:            config.Azure,
			InstanceWatchdog: iwd,
		}

		instanceResource, err = instance.New(c)
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
			Logger: config.Logger,

			Azure:                    config.Azure,
			HostAzureClientSetConfig: config.HostAzureClientSetConfig,
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
		statusResource,
		releaseResource,
		tenantClientsResource,
		namespaceResource,
		serviceResource,
		resourceGroupResource,
		containerURLResource,
		encryptionkeyResource,
		blobObjectResource,
		deploymentResource,
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

	handlesFunc := func(obj interface{}) bool {
		cr, err := key.ToCustomResource(obj)
		if err != nil {
			config.Logger.Log("level", "warning", "message", fmt.Sprintf("invalid object: %s", err), "stack", fmt.Sprintf("%v", err)) // nolint: errcheck
			return false
		}

		if key.OperatorVersion(&cr) == project.Version() {
			return true
		}

		return false
	}

	initCtxFunc := func(ctx context.Context, obj interface{}) (context.Context, error) {
		cr, err := key.ToCustomResource(obj)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		_, vnet, err := net.ParseCIDR(key.VnetCIDR(cr))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		subnets, err := network.Compute(*vnet)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		guestAzureClientSetConfig, err := credential.GetAzureConfig(config.K8sClient.K8sClient(), key.CredentialName(cr), key.CredentialNamespace(cr))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		guestAzureClientSetConfig.EnvironmentName = config.Azure.EnvironmentName

		azureClients, err := client.NewAzureClientSet(*guestAzureClientSetConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		var cloudConfig *cloudconfig.CloudConfig
		{
			c := cloudconfig.Config{
				CertsSearcher:      certsSearcher,
				Logger:             config.Logger,
				RandomkeysSearcher: randomkeysSearcher,

				Azure:        config.Azure,
				AzureConfig:  *guestAzureClientSetConfig,
				AzureNetwork: *subnets,
				Ignition:     config.Ignition,
				OIDC:         config.OIDC,
				SSOPublicKey: config.SSOPublicKey,
			}

			cloudConfig, err = cloudconfig.New(c)
			if err != nil {
				return nil, microerror.Mask(err)
			}
		}

		c := controllercontext.Context{
			AzureClientSet: azureClients,
			AzureNetwork:   subnets,
			CloudConfig:    cloudConfig,
		}
		ctx = controllercontext.NewContext(ctx, c)

		return ctx, nil
	}

	var resourceSet *controller.ResourceSet
	{
		c := controller.ResourceSetConfig{
			Handles:   handlesFunc,
			InitCtx:   initCtxFunc,
			Logger:    config.Logger,
			Resources: resources,
		}

		resourceSet, err = controller.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceSet, nil
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
