package capzcrs

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Name = "capzcrs"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger

	Location string
}

type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger

	location string
}

func New(config Config) (*Resource, error) {
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
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,

		location: config.Location,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}
