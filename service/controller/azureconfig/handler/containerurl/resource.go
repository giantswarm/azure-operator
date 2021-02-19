package containerurl

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsawarefactory"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
)

const (
	Name = "containerurl"
)

type Config struct {
	Logger micrologger.Logger

	MCAzureClientFactory credentialsawarefactory.Interface
}

type Resource struct {
	logger micrologger.Logger

	mcAzureClientFactory credentialsawarefactory.Interface
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.MCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.MCAzureClientFactory must not be empty", config)
	}

	newResource := &Resource{
		logger:               config.Logger,
		mcAzureClientFactory: config.MCAzureClientFactory,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) addContainerURLToContext(ctx context.Context, containerName, storageAccountName, primaryKey string) error {
	r.logger.Debugf(ctx, "setting containerurl to context")

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

	r.logger.Debugf(ctx, "set containerurl to context")

	return nil
}
