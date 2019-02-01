package blobobject

import (
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/v5/blobclient"
)

const (
	// Name is the identifier of the resource.
	Name = "blobobjectv5"
)

type Config struct {
	Logger                micrologger.Logger
	StorageAccountsClient *storage.AccountsClient
}

type Resource struct {
	blobClient blobclient.BlobClient
	logger     micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.StorageAccountsClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.StorageAccountsClient must not be empty", config)
	}

	c := blobclient.Config{
		StorageAccountsClient: config.StorageAccountsClient,
	}

	blobClient, err := blobclient.New(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r := &Resource{
		blobClient: blobClient,
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
