package blobclient

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"
)

const (
	resourceNotFoundError = "NotFound"
	maxRetriesRequests    = 3
)

type Config struct {
	ContainerName          string
	GroupNameFunc          func(v interface{}) (string, error)
	StorageAccountNameFunc func(v interface{}) (string, error)
	StorageAccountsClient  *storage.AccountsClient
}

type BlobClient struct {
	containerName         string
	groupName             string
	storageAccountName    string
	storageAccountsClient *storage.AccountsClient

	// containerURL is configured separately from the default
	// parameters.
	containerURL azblob.ContainerURL
}

func New(config Config) (BlobClient, error) {
	if config.StorageAccountsClient == nil {
		return BlobClient{}, microerror.Maskf(invalidConfigError, "%T.StorageAccountsClient must not be empty", config)
	}

	blobClient := BlobClient{
		storageAccountsClient: config.StorageAccountsClient,
	}

	return blobClient, nil
}

func (c *BlobClient) Boot(ctx context.Context, containerName, groupName, storageAccountName string) error {
	var containerURL azblob.ContainerURL

	c.containerName = containerName
	c.groupName = groupName
	c.storageAccountName = storageAccountName

	key, err := c.getAccountPrimaryKey(ctx)
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
	containerURL = serviceURL.NewContainerURL(containerName)
	_, err = containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	if err != nil {
		return microerror.Mask(err)
	}

	c.containerURL = containerURL

	return nil
}

func (c *BlobClient) BlobExists(ctx context.Context, blobName string) (bool, error) {
	blobURL := c.containerURL.NewBlockBlobURL(blobName)

	_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	if IsBlobNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (c *BlobClient) ContainerExists(ctx context.Context) (bool, error) {
	_, err := c.containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	if IsContainerNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (c *BlobClient) PutBlockBlob(ctx context.Context, blobName string, payload string) (azblob.BlockBlobURL, error) {
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

func (c *BlobClient) GetBlockBlob(ctx context.Context, blobName string) ([]byte, error) {
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

func (c *BlobClient) ListBlobs(ctx context.Context) (*azblob.ListBlobsFlatSegmentResponse, error) {
	var listBlobs *azblob.ListBlobsFlatSegmentResponse

	listBlobs, err := c.containerURL.ListBlobsFlatSegment(
		ctx,
		azblob.Marker{},
		azblob.ListBlobsSegmentOptions{
			Details: azblob.BlobListingDetails{
				Snapshots: false,
			},
		})

	if err != nil {
		return listBlobs, microerror.Mask(err)
	}

	return listBlobs, nil
}

func (c *BlobClient) StorageAccountExists(ctx context.Context, groupName, storageAccountName string) (bool, error) {
	_, err := c.storageAccountsClient.GetProperties(ctx, groupName, storageAccountName)
	if IsStorageAccountNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (c *BlobClient) getAccountPrimaryKey(ctx context.Context) (string, error) {
	keys, err := c.storageAccountsClient.ListKeys(ctx, c.groupName, c.storageAccountName)
	if err != nil {
		return "", microerror.Mask(err)
	}
	if len(*(keys.Keys)) == 0 {
		return "", microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}

	return *(((*keys.Keys)[0]).Value), nil
}
