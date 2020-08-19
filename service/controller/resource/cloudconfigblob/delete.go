package cloudconfigblob

import (
	"context"
	"fmt"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

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

	credentialSecret, err := r.getCredentialSecret(ctx, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	blobName := key.BootstrapBlobName(azureMachinePool)

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting container object %#q", blobName))

	var containerURL azblob.ContainerURL
	{
		containerURL, err = r.getContainerURL(ctx, credentialSecret, key.ClusterID(&azureMachinePool), key.StorageAccountName(&azureMachinePool))
		if IsStorageAccountNotFound(err) {
			// Most probably resource group is already deleted. All good for cloudconfig.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	blob := containerURL.NewBlockBlobURL(blobName)
	_, err = blob.Delete(ctx, azblob.DeleteSnapshotsOptionInclude, azblob.BlobAccessConditions{})
	if blobclient.IsNotFound(err) || blobclient.IsBlobNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "Bootstrap blob not found when trying to delete it")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted container object %#q", blobName))

	return nil
}
