package containerurl

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/v11/controllercontext"
)

const (
	Name = "containerurlv11"
)

type Config struct {
	Logger micrologger.Logger
}

type Resource struct {
	logger micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	newResource := &Resource{
		logger: config.Logger,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) addContainerURLToContext(ctx context.Context, containerName, groupName, storageAccountName, primaryKey string) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "setting containerurl to context")

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	sc, err := azblob.NewSharedKeyCredential(storageAccountName, primaryKey)
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

	r.logger.LogCtx(ctx, "level", "debug", "message", "set containerurl to context")

	return nil
}

func (r *Resource) getStorageAccountsClient(ctx context.Context) (*storage.AccountsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.StorageAccountsClient, nil
}
