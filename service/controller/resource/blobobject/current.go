package blobobject

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/v5/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
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
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil, nil
	}

	containerURL := cc.ContainerURL
	containerName := key.BlobContainerName()
	storageAccountName := key.StorageAccountName(&cr)

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
			return nil, microerror.Mask(err)
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
