package blobobject

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v5/cloudconfig"
	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
)

const (
	blobFormatString      = `https://%s.blob.core.windows.net`
	resourceNotFoundError = "NotFound"
	maxRetriesRequests    = 3
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

type StorageClient struct {
	accountsClient *storage.AccountsClient
	containerURL   azblob.ContainerURL
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

// EnsureDeleted is a noop since the deletion of blob is part is redirected to
// the deletion of container deployment.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (c *StorageClient) BlobExists(ctx context.Context, blobName string) (bool, error) {
	blobURL := c.containerURL.NewBlockBlobURL(blobName)

	_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	if IsBlobNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (c *StorageClient) createBlockBlob(ctx context.Context, blobName string, payload string) (azblob.BlockBlobURL, error) {
	blob := c.containerURL.NewBlockBlobURL(blobName)

	_, err := blob.Upload(
		ctx,
		strings.NewReader(payload),
		azblob.BlobHTTPHeaders{
			ContentType: "text/plain",
		},
		azblob.Metadata{},
		azblob.BlobAccessConditions{},
	)

	return blob, err
}

func (r *Resource) getAccountsClient() (*storage.AccountsClient, error) {
	var accountsClient *storage.AccountsClient

	azureClients, err := client.NewAzureClientSet(r.hostAzureClientSetConfig)
	if err != nil {
		return accountsClient, microerror.Mask(err)
	}
	accountsClient = azureClients.AccountsClient

	return accountsClient, nil
}

func (c *StorageClient) getAccountPrimaryKey(ctx context.Context, accountName, accountGroupName string) (string, error) {
	keys, err := c.accountsClient.ListKeys(ctx, accountGroupName, accountName)
	if err != nil {
		return "", microerror.Mask(err)
	}
	return *(((*keys.Keys)[0]).Value), nil
}

func (c *StorageClient) getBlockBlob(ctx context.Context, blobName string) ([]byte, error) {
	blobURL := c.containerURL.NewBlockBlobURL(blobName)

	response, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	retryReaderOptions := azblob.RetryReaderOptions{
		MaxRetryRequests: maxRetriesRequests,
	}
	defer response.Body(retryReaderOptions).Close()
	blobData, err := ioutil.ReadAll(response.Body(retryReaderOptions))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return blobData, nil
}

func (r *Resource) getCloudConfig(ctx context.Context) (*cloudconfig.CloudConfig, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.CloudConfig, nil
}

func (c *StorageClient) getContainerURL(ctx context.Context, accountName, accountGroupName, containerName string) (azblob.ContainerURL, error) {
	var containerURL azblob.ContainerURL

	key, err := c.getAccountPrimaryKey(ctx, accountName, accountGroupName)
	if err != nil {
		return containerURL, microerror.Mask(err)
	}

	sc, err := azblob.NewSharedKeyCredential(accountName, key)
	if err != nil {
		return containerURL, microerror.Mask(err)
	}

	p := azblob.NewPipeline(sc, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf(blobFormatString, accountName))
	service := azblob.NewServiceURL(*u, p)
	containerURL = service.NewContainerURL(containerName)
	_, err = containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	if IsContainerNotFound(err) {
		return containerURL, nil
	}

	return containerURL, err
}

func (c *StorageClient) listBlobs(ctx context.Context, containerName string) (*azblob.ListBlobsFlatSegmentResponse, error) {
	return c.containerURL.ListBlobsFlatSegment(
		ctx,
		azblob.Marker{},
		azblob.ListBlobsSegmentOptions{
			Details: azblob.BlobListingDetails{
				Snapshots: false,
			},
		})
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
