package cloudconfigblob

import (
	"context"
	"fmt"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	blobName := key.BlobName(&azureMachinePool, key.PrefixWorker())

	var payload string
	{
		payload, err = r.getCloudConfigFromBootstrapSecret(ctx, azureMachinePool)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "bootstrap CR or cloudconfig secret were not found")
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if IsBootstrapCRNotReady(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "bootstrap CR is not ready")
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	var containerURL azblob.ContainerURL
	{
		containerURL, err = r.getContainerURL(ctx, credentialSecret, key.ClusterID(&azureMachinePool), key.StorageAccountName(&azureMachinePool))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring container object %#q contains bootstrap config", blobName))

		_, err = blobclient.PutBlockBlob(ctx, blobName, payload, &containerURL)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured container object %#q contains bootstrap config", blobName))
	}

	return nil
}

func (r *Resource) getCloudConfigFromBootstrapSecret(ctx context.Context, azureMachinePool v1alpha3.AzureMachinePool) (string, error) {
	var sparkCR corev1alpha1.Spark
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Trying to find Bootstrap CR %#q", azureMachinePool.Name))
		err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePool.Namespace, Name: azureMachinePool.Name}, &sparkCR)
		if err != nil {
			return "", microerror.Mask(err)
		}

		if !sparkCR.Status.Ready || sparkCR.Status.DataSecretName == "" {
			return "", microerror.Mask(bootstrapCRNotReady)
		}
	}

	var cloudconfigSecret corev1.Secret
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Trying to find Secret containing bootstrap config %#q", sparkCR.Status.DataSecretName))
		err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePool.Namespace, Name: sparkCR.Status.DataSecretName}, &cloudconfigSecret)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	return string(cloudconfigSecret.Data[key.CloudConfigSecretKey]), nil
}

func (r *Resource) getContainerURL(ctx context.Context, credentialSecret *v1alpha1.CredentialSecret, resourceGroupName, storageAccountName string) (azblob.ContainerURL, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "Finding ContainerURL to upload bootstrap config")

	storageAccountsClient, err := r.clientFactory.GetStorageAccountsClient(credentialSecret.Namespace, credentialSecret.Name)
	if err != nil {
		return azblob.ContainerURL{}, microerror.Mask(err)
	}

	primaryKey, err := r.getPrimaryKey(ctx, storageAccountsClient, resourceGroupName, storageAccountName)
	if err != nil {
		return azblob.ContainerURL{}, microerror.Mask(err)
	}

	sc, err := azblob.NewSharedKeyCredential(storageAccountName, primaryKey)
	if err != nil {
		return azblob.ContainerURL{}, microerror.Mask(err)
	}

	p := azblob.NewPipeline(sc, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", storageAccountName))
	serviceURL := azblob.NewServiceURL(*u, p)
	return serviceURL.NewContainerURL(key.BlobContainerName()), nil
}

func (r *Resource) getPrimaryKey(ctx context.Context, storageAccountsClient *storage.AccountsClient, resourceGroupName, storageAccountName string) (string, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "Finding PrimaryKey for encryption in Storage Account")

	_, err := storageAccountsClient.GetProperties(ctx, resourceGroupName, storageAccountName, storage.AccountExpandGeoReplicationStats)
	if IsStorageAccountNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not find storage account")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return "", nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	keys, err := storageAccountsClient.ListKeys(ctx, resourceGroupName, storageAccountName, "")
	if IsStorageAccountNotProvisioned(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "found storage account is not provisioned")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return "", nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if len(*(keys.Keys)) == 0 {
		return "", microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}

	return *(((*keys.Keys)[0]).Value), nil
}
