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

	storageAccountName := key.StorageAccountName(customObject)

	containerName := key.BlobContainerName()

	output := []ContainerObjectState{}

	{
		b, err := ctlCtx.CloudConfig.NewMasterCloudConfig(customObject)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(customObject, prefixMaster)
		containerObjectState := ContainerObjectState{
			Body:               b,
			ContainerName:      containerName,
			Key:                k,
			StorageAccountName: storageAccountName,
		}

		output = append(output, containerObjectState)
	}

	{
		b, err := ctlCtx.CloudConfig.NewWorkerCloudConfig(customObject)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		k := key.BlobName(customObject, prefixWorker)
		containerObjectState := ContainerObjectState{
			Body:               b,
			ContainerName:      containerName,
			Key:                k,
			StorageAccountName: storageAccountName,
		}

		output = append(output, containerObjectState)
	}

	return output, nil
}
