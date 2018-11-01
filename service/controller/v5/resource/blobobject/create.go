package blobobject

import (
	"context"
	"fmt"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/microerror"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	containerObjectToCreate, err := toContainerObjectState(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	storageAccountsClient, err := r.getAccountsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	sc := &BlobClient{
		storageAccountsClient: storageAccountsClient,
	}

	groupName := key.ClusterID(customObject)
	storageAccountName := key.StorageAccountName(customObject)
	containerName := key.BlobContainerName()

	// if there is no storage account - return and wait for deployment to finish storage account operation.
	_, err = sc.storageAccountsClient.GetProperties(ctx, groupName, storageAccountName)
	if IsStorageAccountNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "blob object's storage account not found, no current objects present")
		return nil
	}
	if err != nil {
		return microerror.Mask(err)
	}

	// if there is no container account - return and wait for deployment to finish container operation.
	containerURL, err := sc.getContainerURL(ctx, storageAccountName, groupName, containerName)
	if IsContainerNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "blob object's container not found, no current objects present")
		return nil
	}
	if err != nil {
		return microerror.Mask(err)
	}
	sc.containerURL = containerURL

	for key, containerObject := range containerObjectToCreate {
		if containerObject.Key != "" {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating container object %#q", key))

			_, err := sc.createBlockBlob(ctx, key, containerObject.Body)
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

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentContainerObject, err := toContainerObjectState(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredContainerObject, err := toContainerObjectState(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the container objects should be created")

	createState := map[string]ContainerObjectState{}

	for key, containerObject := range desiredContainerObject {
		_, ok := currentContainerObject[key]
		if !ok {
			// The desired object does not exist in the current state of the system,
			// so we want to create it.
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("container object %#q should be created", key))
			createState[key] = containerObject
		} else {
			// The desired object exists in the current state of the system, so we do
			// not want to create it. We do track it using an empty object reference
			// though, in order to get some more useful logging in ApplyCreateChange.
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("container object %#q should not be created", key))
			createState[key] = ContainerObjectState{}
		}
	}

	return createState, nil
}
