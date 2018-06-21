package instance

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v3/cloudconfig"
)

const (
	Name = "instancev3"
)

type Config struct {
	CloudConfig *cloudconfig.CloudConfig
	Logger      micrologger.Logger

	Azure           setting.Azure
	AzureConfig     client.AzureClientSetConfig
	TemplateVersion string
}

type Resource struct {
	cloudConfig *cloudconfig.CloudConfig
	logger      micrologger.Logger

	azure           setting.Azure
	azureConfig     client.AzureClientSetConfig
	templateVersion string
}

func New(config Config) (*Resource, error) {
	if config.CloudConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CloudConfig must not be empty", config)
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
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateVersion must not be empty", config)
	}

	r := &Resource{
		cloudConfig: config.CloudConfig,
		logger:      config.Logger,

		azure:           config.Azure,
		azureConfig:     config.AzureConfig,
		templateVersion: config.TemplateVersion,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getDeploymentsClient() (*azureresource.DeploymentsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.DeploymentsClient, nil
}

func (r *Resource) getScaleSetsClient() (*compute.VirtualMachineScaleSetsClient, error) {
	cs, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cs.VirtualMachineScaleSetsClient, nil
}

func (r *Resource) getVMsClient() (*compute.VirtualMachineScaleSetVMsClient, error) {
	cs, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cs.VirtualMachineScaleSetVMsClient, nil
}
