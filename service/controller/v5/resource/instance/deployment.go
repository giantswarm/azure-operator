package instance

import (
	"context"
	"encoding/base64"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v5/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/azure-operator/service/controller/v5/templates"
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
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS(),
			VMSize:          m.VMSize,
		}
		masterNodes = append(masterNodes, n)
	}

	var workerNodes []node
	for _, w := range obj.Spec.Azure.Workers {
		n := node{
			AdminUsername:   key.AdminUsername(obj),
			AdminSSHKeyData: key.AdminSSHKeyData(obj),
			OSImage:         newNodeOSImageCoreOS(),
			VMSize:          w.VMSize,
		}
		workerNodes = append(workerNodes, n)
	}

	containerName := key.BlobContainerName()
	groupName := key.ResourceGroupName(obj)
	storageAccountName := key.StorageAccountName(obj)
	masterBlobName := key.BlobName(obj, key.PrefixMaster())
	workerBlobName := key.BlobName(obj, key.PrefixWorker())

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
		BlobURL: masterBlobURL,
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
		BlobURL: workerBlobURL,
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
		"workerCloudConfigData": encodedWorkerCloudConfig,
		"workerNodes":           workerNodes,
		"workerSubnetID":        cc.WorkerSubnetID,
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
