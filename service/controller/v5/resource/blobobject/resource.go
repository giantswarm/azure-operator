package blobobject

import (
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/giantswarm/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	// Name is the identifier of the resource.
	Name = "blobobjectv5"
)

type Config struct {
	CertsSearcher         certs.Interface
	Logger                micrologger.Logger
	StorageAccountsClient *storage.AccountsClient
}

type Resource struct {
	certsSearcher certs.Interface
	logger        micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CertsSearcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CertsSearcher must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		certsSearcher: config.CertsSearcher,
		logger:        config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func toContainerObjectState(v interface{}) ([]ContainerObjectState, error) {
	if v == nil {
		return nil, nil
	}

	containerObjectState, ok := v.([]ContainerObjectState)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", containerObjectState, v)
	}

	return containerObjectState, nil
}

func objectInSliceByKey(obj ContainerObjectState, list []ContainerObjectState) bool {
	for _, item := range list {
		if obj.Key == item.Key {
			return true
		}
	}
	return false
}

func objectInSliceByKeyAndBody(obj ContainerObjectState, list []ContainerObjectState) bool {
	for _, item := range list {
		if obj.Key == item.Key && obj.Body == item.Body {
			return true
		}
	}
	return false
}
