package containerurl

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding storage account") // nolint: errcheck

	containerName := key.BlobContainerName()
	groupName := key.ClusterID(cr)
	storageAccountName := key.StorageAccountName(cr)

	storageAccountsClient, err := r.getStorageAccountsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = storageAccountsClient.GetProperties(ctx, groupName, storageAccountName, storage.AccountExpandGeoReplicationStats)
	if IsStorageAccountNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not find storage account") // nolint: errcheck
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource") // nolint: errcheck
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	keys, err := storageAccountsClient.ListKeys(ctx, groupName, storageAccountName, "")
	if IsStorageAccountNotProvisioned(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "found storage account is not provisioned") // nolint: errcheck
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource") // nolint: errcheck
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if len(*(keys.Keys)) == 0 {
		return microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}
	primaryKey := *(((*keys.Keys)[0]).Value)

	r.logger.LogCtx(ctx, "level", "debug", "message", "found storage account") // nolint: errcheck
	err = r.addContainerURLToContext(ctx, containerName, storageAccountName, primaryKey)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
