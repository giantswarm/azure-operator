package instance

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/azure-operator/pkg/helpers/vmss"
	"github.com/giantswarm/azure-operator/pkg/project"
	"github.com/giantswarm/azure-operator/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r Resource) newDeployment(ctx context.Context, obj providerv1alpha1.AzureConfig, overwrites map[string]interface{}, location string) (azureresource.Deployment, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}
	err = cc.Validate()
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	prefixWorker := key.PrefixWorker()

	workerBlobName := key.BlobName(obj, prefixWorker)
	cloudConfigURLs := []string{
		workerBlobName,
	}

	for _, key := range cloudConfigURLs {
		blobURL := cc.ContainerURL.NewBlockBlobURL(key)
		_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
		// if blob is not ready - stop instance resource reconciliation
		if err != nil {
			return azureresource.Deployment{}, microerror.Mask(err)
		}
	}

	certificateEncryptionSecretName := key.CertificateEncryptionSecretName(obj)
	encrypter, err := r.getEncrypterObject(ctx, certificateEncryptionSecretName)
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "encryptionkey secret is not found", "secretname", certificateEncryptionSecretName)
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return azureresource.Deployment{}, nil
	} else if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	encryptionKey := encrypter.GetEncryptionKey()
	initialVector := encrypter.GetInitialVector()

	storageAccountsClient, err := r.getStorageAccountsClient(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	groupName := key.ResourceGroupName(obj)
	storageAccountName := key.StorageAccountName(obj)
	keys, err := storageAccountsClient.ListKeys(ctx, groupName, storageAccountName, "")
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	if len(*(keys.Keys)) == 0 {
		return azureresource.Deployment{}, microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}
	primaryKey := *(((*keys.Keys)[0]).Value)
	containerName := key.BlobContainerName()

	// Workers cloudconfig
	workerBlobURL, err := blobclient.GetBlobURL(workerBlobName, containerName, storageAccountName, primaryKey, cc.ContainerURL)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}
	workerCloudConfig, err := vmss.RenderCloudConfig(workerBlobURL, encryptionKey, initialVector, prefixWorker)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"apiLBBackendPoolID":    cc.APILBBackendPoolID,
		"azureOperatorVersion":  project.Version(),
		"clusterID":             key.ClusterID(obj),
		"etcdLBBackendPoolID":   cc.EtcdLBBackendPoolID,
		"vmssMSIEnabled":        r.azure.MSI.Enabled,
		"workerCloudConfigData": workerCloudConfig,
		"workerNodes":           vmss.GetWorkerNodesConfiguration(obj),
		"workerSubnetID":        cc.WorkerSubnetID,
		"zones":                 key.AvailabilityZones(obj, location),
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			TemplateLink: &azureresource.TemplateLink{
				URI:            to.StringPtr(key.ARMTemplateURI(r.templateVersion, "instance", "main.json")),
				ContentVersion: to.StringPtr(key.TemplateContentVersion),
			},
		},
	}

	return d, nil
}
