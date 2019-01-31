package blobobject

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/v5/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// if there is no storage account - return and wait for deployment to finish storage account operation.
	storageAccountExists, err := r.blobClient.StorageAccountExists(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if !storageAccountExists {
		r.logger.LogCtx(ctx, "level", "debug", "message", "blob object's storage account not found, no current objects present")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")
		return nil, nil
	}

	err = r.blobClient.Boot(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding blob object's container")
	// if here is no container account - return and wait for deployment to finish container operation.
	_, err = r.blobClient.ContainerExists(ctx)
	containerExists, err := r.blobClient.ContainerExists(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if !containerExists {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not find blob object's container")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found blob object's container")

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding container objects")

	listBlobs, err := r.blobClient.ListBlobs(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	storageAccountName, err := key.ToStorageAccountName(customObject)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	output := map[string]ContainerObjectState{}
	for _, object := range listBlobs.Segment.BlobItems {
		body, err := r.blobClient.GetBlockBlob(ctx, object.Name)

		if err != nil {
			return output, microerror.Mask(err)
		}

		output[object.Name] = ContainerObjectState{
			Body:               string(body),
			ContainerName:      key.BlobContainerName(),
			Key:                object.Name,
			StorageAccountName: storageAccountName,
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found container objects")

	return output, nil
}
