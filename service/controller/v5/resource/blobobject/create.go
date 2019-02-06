package blobobject

import (
	"context"
	"fmt"

	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v5/blobclient"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	containerURL := cc.ContainerURL

	containerObjectToCreate, err := toContainerObjectState(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, containerObject := range containerObjectToCreate {
		if containerObject.Key != "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating container object %#q", containerObject.Key))

			_, err := blobclient.PutBlockBlob(ctx, containerObject.Key, containerObject.Body, containerURL)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created container object %#q", containerObject.Key))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not create container object %#q", containerObject.Key))
			r.logger.LogCtx(ctx, "level", "debug", "message", "container object already exists")
		}
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentContainerObjects, err := toContainerObjectState(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredContainerObjects, err := toContainerObjectState(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the container objects should be created")

	createState := []ContainerObjectState{}

	for _, desiredContainerObject := range desiredContainerObjects {
		for _, currentContainerObject := range currentContainerObjects {
			if currentContainerObject.Key == desiredContainerObject.Key {
				// The desired object exists in the current state of the system, so we do
				// not want to create it. We do track it using an empty object reference
				// though, in order to get some more useful logging in ApplyCreateChange.
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("container object %#q should not be created", currentContainerObject.Key))
				createState = append(createState, ContainerObjectState{})
				break
			} else {
				// The desired object does not exist in the current state of the system,
				// so we want to create it.
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("container object %#q should be created", currentContainerObject.Key))
				createState = append(createState, currentContainerObject)
			}
		}
	}

	return createState, nil
}
