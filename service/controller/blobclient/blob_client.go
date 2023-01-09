package blobclient

import (
	"context"
	"fmt"
	"io/ioutil" // nolint:staticcheck
	"strings"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"
)

const (
	blobSASValidityTime = 4320
	maxRetriesRequests  = 3
)

func ContainerExists(ctx context.Context, containerURL *azblob.ContainerURL) (bool, error) {
	_, err := containerURL.GetProperties(ctx, azblob.LeaseAccessConditions{})
	if IsContainerNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func PutBlockBlob(ctx context.Context, blobName string, payload string, containerURL *azblob.ContainerURL) (azblob.BlockBlobURL, error) {
	blob := containerURL.NewBlockBlobURL(blobName)

	_, err := blob.Upload(
		ctx,
		strings.NewReader(payload),
		azblob.BlobHTTPHeaders{
			ContentType: "text/plain",
		},
		azblob.Metadata{},
		azblob.BlobAccessConditions{},
		azblob.DefaultAccessTier,
		nil,
		azblob.ClientProvidedKeyOptions{},
		azblob.ImmutabilityPolicyOptions{},
	)
	if err != nil {
		return azblob.BlockBlobURL{}, microerror.Mask(err)
	}

	return blob, nil
}

func GetBlockBlob(ctx context.Context, blobName string, containerURL *azblob.ContainerURL) ([]byte, error) {
	blobURL := containerURL.NewBlockBlobURL(blobName)

	response, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	retryReaderOptions := azblob.RetryReaderOptions{
		MaxRetryRequests: maxRetriesRequests,
	}
	defer response.Body(retryReaderOptions).Close() // nolint:errcheck
	blobData, err := ioutil.ReadAll(response.Body(retryReaderOptions))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return blobData, nil
}

func GetBlobURL(blobName, containerName, storageAccountName, primaryKey string, containerURL *azblob.ContainerURL) (string, error) {

	sharedKeyCredential, err := azblob.NewSharedKeyCredential(storageAccountName, primaryKey)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Set the desired SAS signature values and sign them
	//with the shared key credentials to get the SAS query parameters.
	sasQueryParams, err := azblob.BlobSASSignatureValues{
		BlobName:      blobName,
		ContainerName: containerName,
		ExpiryTime:    time.Now().UTC().Add(blobSASValidityTime * time.Hour),
		// give readonly access
		Permissions: azblob.BlobSASPermissions{Add: false, Read: true, Write: false}.String(),
		Protocol:    azblob.SASProtocolHTTPS,
	}.NewSASQueryParameters(sharedKeyCredential)
	if err != nil {
		return "", microerror.Mask(err)
	}

	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s?%s", storageAccountName, containerName, blobName, sasQueryParams.Encode())

	return blobURL, nil
}

func ListBlobs(ctx context.Context, containerURL *azblob.ContainerURL) (*azblob.ListBlobsFlatSegmentResponse, error) {
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
		return nil, microerror.Mask(err)
	}

	return listBlobs, nil
}
