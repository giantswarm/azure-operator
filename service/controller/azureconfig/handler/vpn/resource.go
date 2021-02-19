package vpn

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsawarefactory"
	"github.com/giantswarm/azure-operator/v5/service/controller/debugger"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"
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

	Azure                setting.Azure
	WCAzureClientFactory credentialsawarefactory.Interface
}

// Resource ensures Microsoft Virtual Network Gateways are running.
type Resource struct {
	ctrlClient ctrlclient.Client
	debugger   *debugger.Debugger
	logger     micrologger.Logger

	azure                setting.Azure
	wcAzureClientFactory credentialsawarefactory.Interface
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
	if config.WCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	r := &Resource{
		ctrlClient: config.CtrlClient,
		debugger:   config.Debugger,
		logger:     config.Logger,

		azure:                config.Azure,
		wcAzureClientFactory: config.WCAzureClientFactory,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
