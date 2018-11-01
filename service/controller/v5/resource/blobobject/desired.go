package blobobject

import (
	"context"
	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/microerror"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	storageAccountName := key.StorageAccountName(customObject)
	containerName := key.BlobContainerName()

	output := map[string]ContainerObjectState{}

	{
		b, err := cc.CloudConfig.NewMasterCloudConfig(customObject)
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
		b, err := cc.CloudConfig.NewWorkerCloudConfig(customObject)
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
