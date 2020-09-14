package workermigration

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	azureclient "github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/workermigration/internal/azure"
)

const (
	Name = "workermigration"
)

type Config struct {
	ClientFactory *azureclient.Factory
	CtrlClient    client.Client
	Logger        micrologger.Logger

	Location string
}

type Resource struct {
	azureapi   azure.API
	ctrlClient client.Client
	logger     micrologger.Logger

	location string
}

func New(config Config) (*Resource, error) {
	if config.ClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClientFactory must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Location == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Location must not be empty", config)
	}

	newResource := &Resource{
		azureapi:   azure.NewAPI(config.ClientFactory),
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,

		location: config.Location,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}
