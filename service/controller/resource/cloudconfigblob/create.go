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
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

// EnsureCreated will make sure that a blob is saved in the Storage Account containing the cloud config for the node pool.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	if machinePool == nil {
		return microerror.Mask(ownerReferenceNotSet)
	}

	credentialSecret, err := r.getCredentialSecret(ctx, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	blobName := key.BootstrapBlobName(azureMachinePool)

	var payload string
	{
		payload, err = r.getCloudConfigFromBootstrapSecret(ctx, machinePool)
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
		if IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find storage account")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		} else if err != nil {
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

// getCloudConfigFromBootstrapSecret returns the Bootstrap cloud config from the Bootstrap secret.
func (r *Resource) getCloudConfigFromBootstrapSecret(ctx context.Context, machinePool *expcapiv1alpha3.MachinePool) (string, error) {
	var err error
	var bootstrapSecretName string
	{
		bootstrapSecretName, err = r.getBootstrapSecretName(ctx, machinePool)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	var cloudconfigSecret corev1.Secret
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Trying to find Secret containing bootstrap config %#q", bootstrapSecretName))
		err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePool.Namespace, Name: bootstrapSecretName}, &cloudconfigSecret)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	return string(cloudconfigSecret.Data[key.CloudConfigSecretKey]), nil
}

// getBootstrapSecretName will try to find Ignition CRs instead of Spark CRs when Ignition Operator is deployed.
// It tries to find a Bootstrap CR which is named after the MachinePool. We may want to change it so we use `MachinePool.Spec.Template.Spec.Bootstrap`.
func (r *Resource) getBootstrapSecretName(ctx context.Context, machinePool *expcapiv1alpha3.MachinePool) (string, error) {
	var sparkCR corev1alpha1.Spark
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Trying to find Bootstrap CR %#q", machinePool.Name))
		err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: machinePool.Namespace, Name: machinePool.Name}, &sparkCR)
		if err != nil {
			return "", microerror.Mask(err)
		}

		if !sparkCR.Status.Ready || sparkCR.Status.DataSecretName == "" {
			return "", microerror.Mask(bootstrapCRNotReady)
		}
	}

	return sparkCR.Status.DataSecretName, nil
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
	if err != nil {
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
