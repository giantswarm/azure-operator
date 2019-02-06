package blobclient

import (
	"context"
	"io/ioutil"
	"strings"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"
)

const (
	resourceNotFoundError = "NotFound"
	maxRetriesRequests    = 3
)

func BlobExists(ctx context.Context, blobName string, containerURL azblob.ContainerURL) (bool, error) {
	blobURL := containerURL.NewBlockBlobURL(blobName)

	_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
	if IsBlobNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func ContainerExists(ctx context.Context, containerURL azblob.ContainerURL) (bool, error) {
	_, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	if IsContainerNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func PutBlockBlob(ctx context.Context, blobName string, payload string, containerURL azblob.ContainerURL) (azblob.BlockBlobURL, error) {
	blob := containerURL.NewBlockBlobURL(blobName)

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

func GetBlockBlob(ctx context.Context, blobName string, containerURL azblob.ContainerURL) ([]byte, error) {
	blobURL := containerURL.NewBlockBlobURL(blobName)

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

func ListBlobs(ctx context.Context, containerURL azblob.ContainerURL) (*azblob.ListBlobsFlatSegmentResponse, error) {
	var listBlobs *azblob.ListBlobsFlatSegmentResponse

	listBlobs, err := containerURL.ListBlobsFlatSegment(
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
