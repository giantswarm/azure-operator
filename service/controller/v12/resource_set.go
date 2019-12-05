package v12

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v12/cloudconfig"
	"github.com/giantswarm/azure-operator/service/controller/v12/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v12/debugger"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/blobobject"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/containerurl"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/deployment"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/dnsrecord"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/encryptionkey"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/endpoints"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/instance"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/namespace"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/resourcegroup"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/service"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/vpn"
	"github.com/giantswarm/azure-operator/service/controller/v12/resource/vpnconnection"
	"github.com/giantswarm/azure-operator/service/credential"
	"github.com/giantswarm/azure-operator/service/network"
	"github.com/giantswarm/certs"
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
	"k8s.io/client-go/kubernetes"
)

type ResourceSetConfig struct {
	CertsSearcher certs.Interface
	G8sClient     versioned.Interface
	K8sClient     kubernetes.Interface
	Logger        micrologger.Logger

	Azure                    setting.Azure
	HostAzureClientSetConfig client.AzureClientSetConfig
	IgnitionPath             string
	InstallationName         string
	ProjectName              string
	OIDC                     setting.OIDC
	SSOPublicKey             string
	// TemplateVersion is a git branch name to use to get Azure Resource
	// Manager templates from.
	TemplateVersion string
}

func NewResourceSet(config ResourceSetConfig) (*controller.ResourceSet, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	var err error

	var certsSearcher certs.Interface
	{
		c := certs.Config{
			K8sClient: config.K8sClient,
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
			K8sClient: config.K8sClient,
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
			RESTClient:               config.G8sClient.ProviderV1alpha1().RESTClient(),
			TenantCluster:            tenantCluster,
			VersionBundleVersionFunc: key.ToVersionBundleVersion,
		}

		statusResource, err = statusresource.NewResource(c)
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
			K8sClient:   config.K8sClient,
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
			CertsSearcher: config.CertsSearcher,
			K8sClient:     config.K8sClient,
			Logger:        config.Logger,
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
			Debugger: newDebugger,
			Logger:   config.Logger,

			Azure:           config.Azure,
			TemplateVersion: config.TemplateVersion,
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
			K8sClient: config.K8sClient,
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

	var instanceResource resource.Interface
	{
		c := instance.Config{
			Debugger:  newDebugger,
			G8sClient: config.G8sClient,
			K8sClient: config.K8sClient,
			Logger:    config.Logger,

			Azure:           config.Azure,
			TemplateVersion: config.TemplateVersion,
		}

		instanceResource, err = instance.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var namespaceResource resource.Interface
	{
		c := namespace.Config{
			K8sClient: config.K8sClient,
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
			K8sClient: config.K8sClient,
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

			Azure:           config.Azure,
			TemplateVersion: config.TemplateVersion,
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
		namespaceResource,
		serviceResource,
		resourceGroupResource,
		containerURLResource,
		encryptionkeyResource,
		blobObjectResource,
		deploymentResource,
		instanceResource,
		endpointsResource,
		dnsrecordResource,
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
		customObject, err := key.ToCustomObject(obj)
		if err != nil {
			config.Logger.Log("level", "warning", "message", fmt.Sprintf("invalid object: %s", err), "stack", fmt.Sprintf("%v", err))
			return false
		}

		if key.VersionBundleVersion(customObject) == VersionBundle().Version {
			return true
		}

		return false
	}

	initCtxFunc := func(ctx context.Context, obj interface{}) (context.Context, error) {
		azureConfig, err := key.ToCustomObject(obj)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		_, vnet, err := net.ParseCIDR(key.VnetCIDR(azureConfig))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		subnets, err := network.Compute(*vnet)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		guestAzureClientSetConfig, err := credential.GetAzureConfig(config.K8sClient, key.CredentialName(azureConfig), key.CredentialNamespace(azureConfig))
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
				IgnitionPath: config.IgnitionPath,
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
