package instance

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

const (
	Name = "instancev2patch1"
)

type Config struct {
	Logger micrologger.Logger

	Azure       setting.Azure
	AzureConfig client.AzureClientSetConfig
}

type Resource struct {
	logger micrologger.Logger

	azure       setting.Azure
	azureConfig client.AzureClientSetConfig
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureConfig.%s", config, err)
	}

	r := &Resource{
		logger: config.Logger,

		azure:       config.Azure,
		azureConfig: config.AzureConfig,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
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
