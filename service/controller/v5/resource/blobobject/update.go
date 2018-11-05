package blobobject

import (
	"context"
	"fmt"

	"github.com/giantswarm/azure-operator/service/controller/v5/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	containerObjectToUpdate, err := toContainerObjectState(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	storageAccountsClient, err := r.getAccountsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	var blobClient *blobclient.BlobClient
	{
		c := blobclient.Config{
			ContainerName:         key.BlobContainerName(),
			GroupName:             key.ClusterID(customObject),
			StorageAccountName:    key.StorageAccountName(customObject),
			StorageAccountsClient: storageAccountsClient,
		}

		blobClient, err = blobclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = blobClient.Boot(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	for key, containerObject := range containerObjectToUpdate {
		if containerObject.Key != "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating container object %#q", key))

			_, err := blobClient.CreateBlockBlob(ctx, key, containerObject.Body)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("created container object %#q", key))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not create container object %#q", key))
			r.logger.LogCtx(ctx, "level", "debug", "message", "container object already exists")
		}
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	create, err := r.newCreateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := controller.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentContainerObject, err := toContainerObjectState(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredContainerObject, err := toContainerObjectState(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the container objects should be updated")

	updateState := map[string]ContainerObjectState{}

	for key, containerObject := range desiredContainerObject {
		if _, ok := currentContainerObject[key]; !ok {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("container object '%s' should not be updated", key))
			updateState[key] = ContainerObjectState{}
		}

		currentObject := currentContainerObject[key]
		if currentObject.Body != "" && containerObject.Body != currentObject.Body {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("container object %#q should be updated", key))
			updateState[key] = containerObject
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("container object %#q should not be updated", key))
			updateState[key] = ContainerObjectState{}
		}
	}

	return updateState, nil
}
