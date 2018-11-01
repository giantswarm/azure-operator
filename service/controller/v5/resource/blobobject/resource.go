package blobobject

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/giantswarm/azure-operator/client"
)

const (
	// Name is the identifier of the resource.
	Name = "blobobjectv5"
)

type Config struct {
	HostAzureClientSetConfig client.AzureClientSetConfig
	Logger                   micrologger.Logger
}

type Resource struct {
	hostAzureClientSetConfig client.AzureClientSetConfig
	logger                   micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if err := config.HostAzureClientSetConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.HostAzureClientSetConfig.%s", err)
	}

	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		hostAzureClientSetConfig: config.HostAzureClientSetConfig,
		logger:                   config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getAccountsClient() (*storage.AccountsClient, error) {
	var storageAccountsClient *storage.AccountsClient

	azureClients, err := client.NewAzureClientSet(r.hostAzureClientSetConfig)
	if err != nil {
		return storageAccountsClient, microerror.Mask(err)
	}
	storageAccountsClient = azureClients.StorageAccountsClient

	return storageAccountsClient, nil
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
