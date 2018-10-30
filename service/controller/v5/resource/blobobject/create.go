package blobobject

import (
	"context"
	"fmt"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/microerror"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	containerObjectToCreate, err := toContainerObjectState(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	accountsClient, err := r.getAccountsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	sc := &StorageClient{
		accountsClient: accountsClient,
	}

	groupName := key.ClusterID(customObject)
	storageAccountName := key.StorageAccountName(customObject)
	containerName := key.BlobContainerName()

	containerURL, err := sc.getContainerURL(ctx, storageAccountName, groupName, containerName)
	if err != nil {
		return microerror.Mask(err)
	}
	sc.containerURL = containerURL

	for key, containerObject := range containerObjectToCreate {
		if containerObject.Key != "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating container object %#q", key))

			_, err := sc.createBlockBlob(ctx, key, containerObject.Body)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created container object %#q", key))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not create container object %#q", key))
			r.logger.LogCtx(ctx, "level", "debug", "message", "container object already exists")
		}
	}

	return nil
}
