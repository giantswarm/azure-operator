package vpn

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v6/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v6/debugger"
)

const (
	// Name is the identifier of the resource.
	Name = "vpnv6"
)

// Config contains information required by Resource.
type Config struct {
	Debugger *debugger.Debugger
	Logger   micrologger.Logger

	Azure           setting.Azure
	TemplateVersion string
}

// Resource ensures Microsoft Virtual Network Gateways are running.
type Resource struct {
	debugger *debugger.Debugger
	logger   micrologger.Logger

	azure           setting.Azure
	templateVersion string
}

// New validates Config and creates a new Resource with it.
func New(config Config) (*Resource, error) {
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
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
		debugger: config.Debugger,
		logger:   config.Logger,

		azure:           config.Azure,
		templateVersion: config.TemplateVersion,
	}

	return r, nil
}

// Name returns the resource name.
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

func (r *Resource) getVirtualNetworkClient(ctx context.Context) (*network.VirtualNetworksClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.VirtualNetworkClient, nil
}
