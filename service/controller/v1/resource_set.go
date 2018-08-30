package v1

import (
	"fmt"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/controller/resource/metricsresource"
	"github.com/giantswarm/operatorkit/controller/resource/retryresource"
	"github.com/giantswarm/randomkeys"

	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v1/cloudconfig"
	"github.com/giantswarm/azure-operator/service/controller/v1/key"
	"github.com/giantswarm/azure-operator/service/controller/v1/resource/deployment"
	"github.com/giantswarm/azure-operator/service/controller/v1/resource/dnsrecord"
	"github.com/giantswarm/azure-operator/service/controller/v1/resource/endpoints"
	"github.com/giantswarm/azure-operator/service/controller/v1/resource/namespace"
	"github.com/giantswarm/azure-operator/service/controller/v1/resource/resourcegroup"
	"github.com/giantswarm/azure-operator/service/controller/v1/resource/service"
	"github.com/giantswarm/azure-operator/service/controller/v1/resource/vnetpeering"
)

type ResourceSetConfig struct {
	K8sClient    kubernetes.Interface
	K8sExtClient apiextensionsclient.Interface
	Logger       micrologger.Logger

	Azure            setting.Azure
	AzureConfig      client.AzureClientSetConfig
	InstallationName string
	ProjectName      string
	// TemplateVersion is a git branch name to use to get Azure Resource
	// Manager templates from.
	TemplateVersion string
}

func NewResourceSet(config ResourceSetConfig) (*controller.ResourceSet, error) {
	var err error

	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.K8sExtClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sExtClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureConfig.%s", config, err)
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ProjectName must not be empty", config)
	}
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateVersion must not be empty", config)
	}

	var certsSearcher *certs.Searcher
	{
		c := certs.Config{
			K8sClient: config.K8sClient,
			Logger:    config.Logger,
		}

		certsSearcher, err = certs.NewSearcher(c)
		if err != nil {
			return nil, microerror.Maskf(err, "certs.NewSearcher")
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
			return nil, microerror.Maskf(err, "randomkeys.NewSearcher")
		}
	}

	var cloudConfig *cloudconfig.CloudConfig
	{
		c := cloudconfig.Config{
			CertsSearcher:      certsSearcher,
			Logger:             config.Logger,
			RandomkeysSearcher: randomkeysSearcher,

			Azure:       config.Azure,
			AzureConfig: config.AzureConfig,
		}

		cloudConfig, err = cloudconfig.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "cloudconfig.New")
		}
	}

	var resourceGroupResource controller.Resource
	{
		c := resourcegroup.Config{
			Logger: config.Logger,

			Azure:            config.Azure,
			AzureConfig:      config.AzureConfig,
			InstallationName: config.InstallationName,
		}

		ops, err := resourcegroup.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		resourceGroupResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var deploymentResource controller.Resource
	{
		c := deployment.Config{
			Logger: config.Logger,

			Azure:           config.Azure,
			AzureConfig:     config.AzureConfig,
			CloudConfig:     cloudConfig,
			TemplateVersion: config.TemplateVersion,
		}

		ops, err := deployment.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		deploymentResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var dnsrecordResource controller.Resource
	{
		c := dnsrecord.Config{
			Logger: config.Logger,

			AzureConfig: config.AzureConfig,
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

	var endpointsResource controller.Resource
	{
		c := endpoints.Config{
			AzureConfig: config.AzureConfig,
			K8sClient:   config.K8sClient,
			Logger:      config.Logger,
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

	var namespaceResource controller.Resource
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

	var serviceResource controller.Resource
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

	var vnetPeeringResource controller.Resource
	{
		c := vnetpeering.Config{
			Logger: config.Logger,

			Azure:       config.Azure,
			AzureConfig: config.AzureConfig,
		}

		ops, err := vnetpeering.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		vnetPeeringResource, err = toCRUDResource(config.Logger, ops)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	resources := []controller.Resource{
		namespaceResource,
		serviceResource,
		resourceGroupResource,
		deploymentResource,
		endpointsResource,
		dnsrecordResource,
		vnetPeeringResource,
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
		c := metricsresource.WrapConfig{
			Name: config.ProjectName,
		}

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

	var resourceSet *controller.ResourceSet
	{
		c := controller.ResourceSetConfig{
			Handles:   handlesFunc,
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

func toCRUDResource(logger micrologger.Logger, ops controller.CRUDResourceOps) (*controller.CRUDResource, error) {
	c := controller.CRUDResourceConfig{
		Logger: logger,
		Ops:    ops,
	}

	r, err := controller.NewCRUDResource(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r, nil
}
