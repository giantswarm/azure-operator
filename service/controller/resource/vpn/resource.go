package vpn

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

const (
	// Name is the identifier of the resource.
	Name = "vpn"
)

// Config contains information required by Resource.
type Config struct {
	CtrlClient ctrlclient.Client
	Debugger   *debugger.Debugger
	Logger     micrologger.Logger

	Azure setting.Azure
}

// Resource ensures Microsoft Virtual Network Gateways are running.
type Resource struct {
	ctrlClient ctrlclient.Client
	debugger   *debugger.Debugger
	logger     micrologger.Logger

	azure setting.Azure
}

// New validates Config and creates a new Resource with it.
func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	r := &Resource{
		ctrlClient: config.CtrlClient,
		debugger:   config.Debugger,
		logger:     config.Logger,

		azure: config.Azure,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getDeploymentsClient(ctx context.Context) (*azureresource.DeploymentsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.DeploymentsClient, nil
}

func (r *Resource) getVirtualNetworkClient(ctx context.Context) (*network.VirtualNetworksClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.VirtualNetworkClient, nil
}
