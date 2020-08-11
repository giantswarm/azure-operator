package cloudconfig

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	credentialDefaultName = "credential-default"
	credentialNamespace   = "giantswarm"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, &cr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	storageAccountsClient, err := r.azureClientsFactory.GetStorageAccountsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	containerURL, err := r.getContainerURL(ctx, storageAccountsClient, key.ClusterID(&cr), key.BlobContainerName(), key.StorageAccountName(&cr))
	if IsStorageAccountNotProvisioned(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "found storage account is not provisioned")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil, microerror.Mask(err)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

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
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found blob object's container")

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding container objects")

	listBlobs, err := blobclient.ListBlobs(ctx, containerURL)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var output []ContainerObjectState
	for _, object := range listBlobs.Segment.BlobItems {
		body, err := blobclient.GetBlockBlob(ctx, object.Name, containerURL)

		if err != nil {
			return nil, microerror.Mask(err)
		}

		containerObjectState := ContainerObjectState{
			Body:               string(body),
			ContainerName:      key.BlobContainerName(),
			Key:                object.Name,
			StorageAccountName: key.StorageAccountName(&cr),
		}

		output = append(output, containerObjectState)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found container objects")

	return output, nil
}
