package containerurl

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
)

const (
	Name = "containerurlv5"
)

type Config struct {
	Logger                micrologger.Logger
	StorageAccountsClient *storage.AccountsClient
}

type Resource struct {
	logger                micrologger.Logger
	storageAccountsClient *storage.AccountsClient
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.StorageAccountsClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.StorageAccountsClient must not be empty", config)
	}

	newResource := &Resource{
		logger:                config.Logger,
		storageAccountsClient: config.StorageAccountsClient,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) addContainerURLToContext(ctx context.Context, containerName, groupName, storageAccountName string) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	key, err := r.getAccountPrimaryKey(ctx, groupName, storageAccountName)
	if err != nil {
		return microerror.Mask(err)
	}

	sc, err := azblob.NewSharedKeyCredential(storageAccountName, key)
	if err != nil {
		return microerror.Mask(err)
	}

	p := azblob.NewPipeline(sc, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", storageAccountName))
	serviceURL := azblob.NewServiceURL(*u, p)
	containerURL := serviceURL.NewContainerURL(containerName)
	_, err = containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	if err != nil {
		return microerror.Mask(err)
	}

	cc.ContainerURL = &containerURL

	return nil
}

func (r *Resource) storageAccountExists(ctx context.Context, groupName, storageAccountName string) (bool, error) {
	_, err := r.storageAccountsClient.GetProperties(ctx, groupName, storageAccountName)
	if IsStorageAccountNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (r *Resource) getAccountPrimaryKey(ctx context.Context, groupName, storageAccountName string) (string, error) {
	keys, err := r.storageAccountsClient.ListKeys(ctx, groupName, storageAccountName)
	if err != nil {
		return "", microerror.Mask(err)
	}
	if len(*(keys.Keys)) == 0 {
		return "", microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}

	return *(((*keys.Keys)[0]).Value), nil
}
