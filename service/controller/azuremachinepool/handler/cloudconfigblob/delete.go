package cloudconfigblob

import (
	"context"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v8/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

// EnsureDeleted will delete the blob in the Storage Account containing the cloud config for the node pool.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	if machinePool == nil {
		// If MachinePool doesn't exist anymore, there's nothing we can do
		// about it. Returning an error here would just keep finalizer and
		// prevent CR from deletion forever.
		return nil
	}

	blobName := key.BootstrapBlobName(azureMachinePool)

	r.logger.Debugf(ctx, "deleting container object %#q", blobName)

	var containerURL azblob.ContainerURL
	{
		containerURL, err = r.getContainerURL(ctx, &azureMachinePool, key.ClusterID(&azureMachinePool), key.StorageAccountName(&azureMachinePool))
		if IsNotFound(err) {
			// Resource Group deleted or deletion in progress. Storage Account already gone.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	blob := containerURL.NewBlockBlobURL(blobName)
	_, err = blob.Delete(ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
	if blobclient.IsNotFound(err) || blobclient.IsBlobNotFound(err) {
		r.logger.Debugf(ctx, "Bootstrap blob not found when trying to delete it")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "deleted container object %#q", blobName)

	return nil
}
