package instance

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v4patch1/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v4patch1/debugger"
)

const (
	Name = "instancev4patch1"
)

type Config struct {
	Debugger  *debugger.Debugger
	G8sClient versioned.Interface
	Logger    micrologger.Logger

	Azure           setting.Azure
	TemplateVersion string
}

type Resource struct {
	debugger  *debugger.Debugger
	g8sClient versioned.Interface
	logger    micrologger.Logger

	azure           setting.Azure
	templateVersion string
}

func New(config Config) (*Resource, error) {
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateVersion must not be empty", config)
	}

	r := &Resource{
		debugger:  config.Debugger,
		g8sClient: config.G8sClient,
		logger:    config.Logger,

		azure:           config.Azure,
		templateVersion: config.TemplateVersion,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getDeploymentsClient(ctx context.Context) (*azureresource.DeploymentsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.DeploymentsClient, nil
}

func (r *Resource) getScaleSetsClient(ctx context.Context) (*compute.VirtualMachineScaleSetsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualMachineScaleSetsClient, nil
}

func (r *Resource) getVMsClient(ctx context.Context) (*compute.VirtualMachineScaleSetVMsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualMachineScaleSetVMsClient, nil
}
