package instance

import (
	"context"
	"encoding/base64"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/azure-operator/service/controller/v12/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/v12/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"github.com/giantswarm/azure-operator/service/controller/v12/templates"
)

func (r Resource) newDeployment(ctx context.Context, obj providerv1alpha1.AzureConfig, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}
	err = cc.Validate()
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	cloudConfigURLs := []string{
		key.BlobName(obj, key.PrefixMaster()),
		key.BlobName(obj, key.PrefixWorker()),
	}

	for _, key := range cloudConfigURLs {
		blobURL := cc.ContainerURL.NewBlockBlobURL(key)
		_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
		// if blob is not ready - stop instance resource reconciliation
		if err != nil {
			return azureresource.Deployment{}, microerror.Mask(err)
		}
	}

	var masterNodes []node
	for _, m := range obj.Spec.Azure.Masters {
		n := node{
			AdminUsername:       key.AdminUsername(obj),
			AdminSSHKeyData:     key.AdminSSHKeyData(obj),
			OSImage:             newNodeOSImageCoreOS(),
			VMSize:              m.VMSize,
			DockerVolumeSizeGB:  m.DockerVolumeSizeGB,
			KubeletVolumeSizeGB: m.KubeletVolumeSizeGB,
		}
		masterNodes = append(masterNodes, n)
	}

	var workerNodes []node
	for _, w := range obj.Spec.Azure.Workers {
		n := node{
			AdminUsername:       key.AdminUsername(obj),
			AdminSSHKeyData:     key.AdminSSHKeyData(obj),
			OSImage:             newNodeOSImageCoreOS(),
			VMSize:              w.VMSize,
			DockerVolumeSizeGB:  w.DockerVolumeSizeGB,
			KubeletVolumeSizeGB: w.KubeletVolumeSizeGB,
		}
		workerNodes = append(workerNodes, n)
	}

	containerName := key.BlobContainerName()
	groupName := key.ResourceGroupName(obj)
	storageAccountName := key.StorageAccountName(obj)
	masterBlobName := key.BlobName(obj, key.PrefixMaster())
	workerBlobName := key.BlobName(obj, key.PrefixWorker())
	zones := key.AvailabilityZones(obj)

	var vmssTemplateFile = "vmss.json"
	{
		if zones != nil {
			// Setting the AZ parameter on a template that doesn't already have
			// it would fail, so we need to use a different ARM template.
			vmssTemplateFile = "vmss-az.json"
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

	keys, err := storageAccountsClient.ListKeys(ctx, groupName, storageAccountName)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	if len(*(keys.Keys)) == 0 {
		return azureresource.Deployment{}, microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}
	primaryKey := *(((*keys.Keys)[0]).Value)

	masterBlobURL, err := blobclient.GetBlobURL(ctx, masterBlobName, containerName, storageAccountName, primaryKey, cc.ContainerURL)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	c := SmallCloudconfigConfig{
		BlobURL:       masterBlobURL,
		EncryptionKey: encryptionKey,
		InitialVector: initialVector,
		InstanceRole:  key.PrefixMaster(),
	}
	masterCloudConfig, err := templates.Render(key.CloudConfigSmallTemplates(), c)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}
	encodedMasterCloudConfig := base64.StdEncoding.EncodeToString([]byte(masterCloudConfig))

	workerBlobURL, err := blobclient.GetBlobURL(ctx, workerBlobName, containerName, storageAccountName, primaryKey, cc.ContainerURL)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	c = SmallCloudconfigConfig{
		BlobURL:       workerBlobURL,
		EncryptionKey: encryptionKey,
		InitialVector: initialVector,
		InstanceRole:  key.PrefixWorker(),
	}
	workerCloudConfig, err := templates.Render(key.CloudConfigSmallTemplates(), c)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}
	encodedWorkerCloudConfig := base64.StdEncoding.EncodeToString([]byte(workerCloudConfig))

	defaultParams := map[string]interface{}{
		"apiLBBackendPoolID":    cc.APILBBackendPoolID,
		"clusterID":             key.ClusterID(obj),
		"etcdLBBackendPoolID":   cc.EtcdLBBackendPoolID,
		"masterCloudConfigData": encodedMasterCloudConfig,
		"masterNodes":           masterNodes,
		"masterSubnetID":        cc.MasterSubnetID,
		"vmssMSIEnabled":        r.azure.MSI.Enabled,
		"vmssTemplateFile":      vmssTemplateFile,
		"workerCloudConfigData": encodedWorkerCloudConfig,
		"workerNodes":           workerNodes,
		"workerSubnetID":        cc.WorkerSubnetID,
		"zones":                 zones,
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
