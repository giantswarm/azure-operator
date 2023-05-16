package blobobject

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/crud"

	"github.com/giantswarm/azure-operator/v8/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v8/service/controller/controllercontext"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	containerObjectToUpdate, err := toContainerObjectState(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, containerObject := range containerObjectToUpdate {
		r.logger.Debugf(ctx, "updating container object %#q", containerObject.Key)

		_, err := blobclient.PutBlockBlob(ctx, containerObject.Key, containerObject.Body, cc.ContainerURL)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated container object %#q", containerObject.Key)
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	create, err := r.newCreateChange(ctx, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, currentState, desiredState interface{}) (interface{}, error) {
	currentContainerObjects, err := toContainerObjectState(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredContainerObjects, err := toContainerObjectState(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "finding out if the container objects should be updated")

	updateState := []ContainerObjectState{}

	for _, desiredContainerObject := range desiredContainerObjects {
		if objectInSliceByKeyAndBody(desiredContainerObject, currentContainerObjects) {
			r.logger.Debugf(ctx, "container object %#q should not be updated", desiredContainerObject.Key)
		} else {
			r.logger.Debugf(ctx, "container object %#q should be updated", desiredContainerObject.Key)
			updateState = append(updateState, desiredContainerObject)
		}
	}

	return updateState, nil
}
