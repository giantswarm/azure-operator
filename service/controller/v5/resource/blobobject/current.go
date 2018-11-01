package blobobject

import (
	"context"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/microerror"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "looking for container objects")

	storageAccountsClient, err := r.getAccountsClient()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	sc := &BlobClient{
		storageAccountsClient: storageAccountsClient,
	}

	groupName := key.ClusterID(customObject)
	storageAccountName := key.StorageAccountName(customObject)
	containerName := key.BlobContainerName()

	// if there is no storage account - return and wait for deployment to finish storage account operation.
	_, err = sc.storageAccountsClient.GetProperties(ctx, groupName, storageAccountName)
	if IsStorageAccountNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "blob object's storage account not found, no current objects present")
		return nil, nil
	}
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// if there is no container account - return and wait for deployment to finish container operation.
	containerURL, err := sc.getContainerURL(ctx, storageAccountName, groupName, containerName)
	if IsContainerNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "blob object's container not found, no current objects present")
		return nil, nil
	}
	if err != nil {
		return nil, microerror.Mask(err)
	}
	sc.containerURL = containerURL

	r.logger.LogCtx(ctx, "level", "debug", "message", "found the blob container")

	listBlobs, err := sc.listBlobs(ctx, containerName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	output := map[string]ContainerObjectState{}
	for _, object := range listBlobs.Segment.BlobItems {
		body, err := sc.getBlockBlob(ctx, object.Name)

		if err != nil {
			return output, microerror.Mask(err)
		}

		output[object.Name] = ContainerObjectState{
			Body:               string(body),
			ContainerName:      containerName,
			Key:                object.Name,
			StorageAccountName: storageAccountName,
		}
	}

	return output, nil

}
