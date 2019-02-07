package blobobject

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/v5/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if cc.ContainerURL == nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", "containerurl resource is not ready")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	containerURL := cc.ContainerURL
	containerName := key.BlobContainerName()
	storageAccountName := key.StorageAccountName(customObject)

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding blob object's container")
	// if here is no container account - return and wait for deployment to finish container operation.
	containerExists, err := blobclient.ContainerExists(ctx, containerURL)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if !containerExists {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not find blob object's container")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found blob object's container")

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding container objects")

	listBlobs, err := blobclient.ListBlobs(ctx, containerURL)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	output := []ContainerObjectState{}
	for _, object := range listBlobs.Segment.BlobItems {
		body, err := blobclient.GetBlockBlob(ctx, object.Name, containerURL)

		if err != nil {
			return output, microerror.Mask(err)
		}

		containerObjectState := ContainerObjectState{
			Body:               string(body),
			ContainerName:      containerName,
			Key:                object.Name,
			StorageAccountName: storageAccountName,
		}

		output = append(output, containerObjectState)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found container objects")

	return output, nil
}
