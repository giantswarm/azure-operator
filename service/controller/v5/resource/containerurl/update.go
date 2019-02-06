package containerurl

import (
	"context"

	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/microerror"
)

func (r *Resource) EnsureUpdated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	containerName := key.BlobContainerName()
	groupName := key.ClusterID(customObject)
	storageAccountName := key.StorageAccountName(customObject)

	err = r.addContainerURLToContext(ctx, containerName, groupName, storageAccountName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
