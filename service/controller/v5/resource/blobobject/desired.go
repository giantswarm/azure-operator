package blobobject

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ctlCtx, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	storageAccountName, err := key.ToStorageAccountName(customObject)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	containerName := key.BlobContainerName()

	output := map[string]ContainerObjectState{}

	{
		b, err := ctlCtx.CloudConfig.NewMasterCloudConfig(customObject)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(customObject, prefixMaster)
		output[k] = ContainerObjectState{
			Body:               b,
			ContainerName:      containerName,
			Key:                k,
			StorageAccountName: storageAccountName,
		}
	}

	{
		b, err := ctlCtx.CloudConfig.NewWorkerCloudConfig(customObject)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(customObject, prefixWorker)
		output[k] = ContainerObjectState{
			Body:               b,
			ContainerName:      containerName,
			Key:                k,
			StorageAccountName: storageAccountName,
		}
	}

	return output, nil
}
