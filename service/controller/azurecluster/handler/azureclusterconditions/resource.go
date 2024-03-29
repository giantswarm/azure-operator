package azureclusterconditions

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"sigs.k8s.io/controller-runtime/pkg/client"

	azureclient "github.com/giantswarm/azure-operator/v8/client"
)

const (
	// Name is the identifier of the resource.
	Name = "azureclusterconditions"
)

type Config struct {
	AzureClientsFactory *azureclient.OrganizationFactory
	CtrlClient          client.Client
	InstallationName    string
	Logger              micrologger.Logger
}

// Resource ensures that AzureCluster Status Conditions are set.
type Resource struct {
	azureClientsFactory *azureclient.OrganizationFactory
	ctrlClient          client.Client
	installationName    string
	logger              micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstallationName must not be empty", config)
	}
	if config.AzureClientsFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureClientsFactory must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		azureClientsFactory: config.AzureClientsFactory,
		ctrlClient:          config.CtrlClient,
		installationName:    config.InstallationName,
		logger:              config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
