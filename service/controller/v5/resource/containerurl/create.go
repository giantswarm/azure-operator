package containerurl

import (
	"context"

	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	containerName := key.BlobContainerName()
	groupName := key.ClusterID(customObject)
	storageAccountName := key.StorageAccountName(customObject)

	storageAccountExists, err := r.storageAccountExists(ctx, groupName, storageAccountName)
	if err != nil {
		return microerror.Mask(err)
	}

	if !storageAccountExists {
		r.logger.LogCtx(ctx, "level", "debug", "message", "storage account not found")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")
		return nil
	}

	err = r.addContainerURLToContext(ctx, containerName, groupName, storageAccountName)
	if IsStorageAccountNotProvisioned(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "storage account not provisioned")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")
		return nil

	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
