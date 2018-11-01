package blobobject

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/giantswarm/microerror"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
)

const (
	blobFormatString      = `https://%s.blob.core.windows.net`
	resourceNotFoundError = "NotFound"
	maxRetriesRequests    = 3
)

type BlobClient struct {
	storageAccountsClient *storage.AccountsClient
	containerURL          azblob.ContainerURL
}

func (c *BlobClient) BlobExists(ctx context.Context, blobName string) (bool, error) {
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

func (c *BlobClient) createBlockBlob(ctx context.Context, blobName string, payload string) (azblob.BlockBlobURL, error) {
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

func (c *BlobClient) getAccountPrimaryKey(ctx context.Context, accountName, accountGroupName string) (string, error) {
	keys, err := c.storageAccountsClient.ListKeys(ctx, accountGroupName, accountName)
	if err != nil {
		return "", microerror.Mask(err)
	}
	return *(((*keys.Keys)[0]).Value), nil
}

func (c *BlobClient) getBlockBlob(ctx context.Context, blobName string) ([]byte, error) {
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

func (c *BlobClient) getContainerURL(ctx context.Context, accountName, accountGroupName, containerName string) (azblob.ContainerURL, error) {
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

func (c *BlobClient) listBlobs(ctx context.Context, containerName string) (*azblob.ListBlobsFlatSegmentResponse, error) {
	return c.containerURL.ListBlobsFlatSegment(
		ctx,
		azblob.Marker{},
		azblob.ListBlobsSegmentOptions{
			Details: azblob.BlobListingDetails{
				Snapshots: false,
			},
		})
}
