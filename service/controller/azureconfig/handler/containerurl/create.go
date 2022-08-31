package containerurl

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/v6/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "finding storage account")

	containerName := key.BlobContainerName()
	groupName := key.ClusterID(&cr)
	storageAccountName := key.StorageAccountName(&cr)

	storageAccountsClient, err := r.getStorageAccountsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = storageAccountsClient.GetProperties(ctx, groupName, storageAccountName, storage.AccountExpandGeoReplicationStats)
	if IsStorageAccountNotFound(err) {
		r.logger.Debugf(ctx, "did not find storage account")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	keys, err := storageAccountsClient.ListKeys(ctx, groupName, storageAccountName, "")
	if IsStorageAccountNotProvisioned(err) {
		r.logger.Debugf(ctx, "found storage account is not provisioned")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if len(*(keys.Keys)) == 0 {
		return microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}
	primaryKey := *(((*keys.Keys)[0]).Value)

	r.logger.Debugf(ctx, "found storage account")
	err = r.addContainerURLToContext(ctx, containerName, storageAccountName, primaryKey)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
