package containerurl

import (
	"context"

	"github.com/giantswarm/azure-operator/service/controller/v8/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding storage account")

	containerName := key.BlobContainerName()
	groupName := key.ClusterID(customObject)
	storageAccountName := key.StorageAccountName(customObject)

	storageAccountsClient, err := r.getStorageAccountsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = storageAccountsClient.GetProperties(ctx, groupName, storageAccountName)
	if IsStorageAccountNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not find storage account")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	keys, err := storageAccountsClient.ListKeys(ctx, groupName, storageAccountName)
	if IsStorageAccountNotProvisioned(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "found storage account is not provisioned")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if len(*(keys.Keys)) == 0 {
		return microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}
	primaryKey := *(((*keys.Keys)[0]).Value)

	r.logger.LogCtx(ctx, "level", "debug", "message", "found storage account")
	err = r.addContainerURLToContext(ctx, containerName, groupName, storageAccountName, primaryKey)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
