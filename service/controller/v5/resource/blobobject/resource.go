package blobobject

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/v5/blobclient"
)

const (
	// Name is the identifier of the resource.
	Name = "blobobjectv5"
)

type Config struct {
	BlobClient blobclient.BlobClient
	Logger     micrologger.Logger
}

type Resource struct {
	blobClient blobclient.BlobClient
	logger     micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		blobClient: config.BlobClient,
		logger:     config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func toContainerObjectState(v interface{}) (map[string]ContainerObjectState, error) {
	if v == nil {
		return nil, nil
	}

	containerObjectState, ok := v.(map[string]ContainerObjectState)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", containerObjectState, v)
	}

	return containerObjectState, nil
}
