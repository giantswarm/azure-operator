package v1

import (
	"fmt"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"
	"github.com/giantswarm/randomkeys"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/service/azureconfig/v1/cloudconfig"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/key"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/resource/deployment"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/resource/dnsrecord"
	"github.com/giantswarm/azure-operator/service/azureconfig/v1/resource/resourcegroup"
)

const (
	ResourceRetries uint64 = 3
)

type ResourceSetConfig struct {
	K8sClient    kubernetes.Interface
	K8sExtClient apiextensionsclient.Interface
	Logger       micrologger.Logger

	AzureConfig client.AzureConfig
	ProjectName string
	// TemplateVersion is a git branch name to use to get Azure Resource
	// Manager templates from.
	TemplateVersion string
}

func NewResourceSet(config ResourceSetConfig) (*framework.ResourceSet, error) {
	var err error

	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sClient must not be empty")
	}
	if config.K8sExtClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.K8sExtClient must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig.%s", err)
	}
	if config.ProjectName == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.ProjectName must not be empty")
	}
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TemplateVersion must not be empty")
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

			AzureConfig: config.AzureConfig,
		}

		cloudConfig, err = cloudconfig.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "cloudconfig.New")
		}
	}

	var resourceGroupResource *resourcegroup.Resource
	{
		c := resourcegroup.Config{
			AzureConfig: config.AzureConfig,
			Logger:      config.Logger,
		}

		resourceGroupResource, err = resourcegroup.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "resourcegroup.New")
		}
	}

	var deploymentResource *deployment.Resource
	{
		c := deployment.Config{
			CertsSearcher: certsSearcher,
			Logger:        config.Logger,

			AzureConfig:     config.AzureConfig,
			CloudConfig:     cloudConfig,
			TemplateVersion: config.TemplateVersion,
		}

		deploymentResource, err = deployment.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "deployment.New")
		}
	}

	var dnsrecordResource *dnsrecord.Resource
	{
		c := dnsrecord.Config{
			Logger: config.Logger,

			AzureConfig: config.AzureConfig,
		}

		dnsrecordResource, err = dnsrecord.New(c)
		if err != nil {
			return nil, microerror.Maskf(err, "dnsrecord.New")
		}
	}

	resources := []framework.Resource{
		resourceGroupResource,
		deploymentResource,
		dnsrecordResource,
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

	var resourceSet *framework.ResourceSet
	{
		c := framework.ResourceSetConfig{
			Handles:   handlesFunc,
			Logger:    config.Logger,
			Resources: resources,
		}

		resourceSet, err = framework.NewResourceSet(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return resourceSet, nil
}
