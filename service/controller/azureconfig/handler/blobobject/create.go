package blobobject

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v8/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v8/service/controller/controllercontext"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	containerObjectToCreate, err := toContainerObjectState(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, containerObject := range containerObjectToCreate {
		r.logger.Debugf(ctx, "creating container object %#q", containerObject.Key)

		_, err := blobclient.PutBlockBlob(ctx, containerObject.Key, containerObject.Body, cc.ContainerURL)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "created container object %#q", containerObject.Key)
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, currentState, desiredState interface{}) (interface{}, error) {
	currentContainerObjects, err := toContainerObjectState(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredContainerObjects, err := toContainerObjectState(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "finding out if the container objects should be created")

	createState := []ContainerObjectState{}

	for _, desiredContainerObject := range desiredContainerObjects {
		if objectInSliceByKey(desiredContainerObject, currentContainerObjects) {
			r.logger.Debugf(ctx, "container object %#q should not be created", desiredContainerObject.Key)
		} else {
			r.logger.Debugf(ctx, "container object %#q should be created", desiredContainerObject.Key)
			createState = append(createState, desiredContainerObject)
		}
	}

	return createState, nil
}
