package azureclusteridentity

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsaware"
)

const (
	// Name is the identifier of the resource.
	Name = "azureclusteridentity"
)

type Config struct {
	WCAzureClientsFactory credentialsaware.Factory
	CtrlClient            ctrlclient.Client
	Logger                micrologger.Logger
}

type Resource struct {
	wcAzureClientsFactory credentialsaware.Factory
	ctrlClient            ctrlclient.Client
	logger                micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		wcAzureClientsFactory: config.WCAzureClientsFactory,
		ctrlClient:            config.CtrlClient,
		logger:                config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
