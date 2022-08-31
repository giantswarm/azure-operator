package azureclusteridentity

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v6/client"
)

const (
	// Name is the identifier of the resource.
	Name = "azureclusteridentity"
)

type Config struct {
	AzureClientsFactory client.OrganizationFactory
	CtrlClient          ctrlclient.Client
	Logger              micrologger.Logger
}

type Resource struct {
	azureClientsFactory client.OrganizationFactory
	ctrlClient          ctrlclient.Client
	logger              micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		azureClientsFactory: config.AzureClientsFactory,
		ctrlClient:          config.CtrlClient,
		logger:              config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
