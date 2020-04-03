package instance

import (
	"context"
	"encoding/base64"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/azure-operator/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/templates"
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

	prefixMaster := key.PrefixMaster()
	prefixWorker := key.PrefixWorker()

	masterBlobName := key.BlobName(obj, prefixMaster)
	workerBlobName := key.BlobName(obj, prefixWorker)
	cloudConfigURLs := []string{
		masterBlobName,
		workerBlobName,
	}

	var distroVersion string
	{
		for _, component := range cc.Release.Components {
			if component.Name == key.ContainerLinuxComponentName {
				distroVersion = component.Version
				break
			}
		}
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
		r.logger.LogCtx(ctx, "level", "debug", "message", "encryptionkey secret is not found", "secretname", certificateEncryptionSecretName) // nolint: errcheck
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource") // nolint: errcheck
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

	// Masters cloudconfig
	masterBlobURL, err := blobclient.GetBlobURL(masterBlobName, containerName, storageAccountName, primaryKey, cc.ContainerURL)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}
	masterCloudConfig, err := renderCloudConfig(masterBlobURL, encryptionKey, initialVector, prefixMaster)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	// Workers cloudconfig
	workerBlobURL, err := blobclient.GetBlobURL(workerBlobName, containerName, storageAccountName, primaryKey, cc.ContainerURL)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}
	workerCloudConfig, err := renderCloudConfig(workerBlobURL, encryptionKey, initialVector, prefixWorker)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	defaultParams := map[string]interface{}{
		"apiLBBackendPoolID":    cc.APILBBackendPoolID,
		"clusterID":             key.ClusterID(obj),
		"etcdLBBackendPoolID":   cc.EtcdLBBackendPoolID,
		"masterCloudConfigData": masterCloudConfig,
		"masterNodes":           getMasterNodesConfiguration(obj, distroVersion),
		"masterSubnetID":        cc.MasterSubnetID,
		"vmssMSIEnabled":        r.azure.MSI.Enabled,
		"workerCloudConfigData": workerCloudConfig,
		"workerNodes":           getWorkerNodesConfiguration(obj, distroVersion),
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

func renderCloudConfig(blobURL string, encryptionKey string, initialVector string, instanceRole string) (string, error) {
	smallCloudconfigConfig := SmallCloudconfigConfig{
		BlobURL:       blobURL,
		EncryptionKey: encryptionKey,
		InitialVector: initialVector,
		InstanceRole:  instanceRole,
	}
	cloudConfig, err := templates.Render(key.CloudConfigSmallTemplates(), smallCloudconfigConfig)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return base64.StdEncoding.EncodeToString([]byte(cloudConfig)), nil
}

func getMasterNodesConfiguration(obj providerv1alpha1.AzureConfig, distroVersion string) []node {
	return getNodesConfiguration(key.AdminUsername(obj), key.AdminSSHKeyData(obj), distroVersion, obj.Spec.Azure.Masters)
}

func getWorkerNodesConfiguration(obj providerv1alpha1.AzureConfig, distroVersion string) []node {
	return getNodesConfiguration(key.AdminUsername(obj), key.AdminSSHKeyData(obj), distroVersion, obj.Spec.Azure.Workers)
}

func getNodesConfiguration(adminUsername string, adminSSHKeyData string, distroVersion string, nodesSpecs []providerv1alpha1.AzureConfigSpecAzureNode) []node {
	var nodes []node
	for _, m := range nodesSpecs {
		n := newNode(adminUsername, adminSSHKeyData, distroVersion, m.VMSize, m.DockerVolumeSizeGB, m.KubeletVolumeSizeGB)
		nodes = append(nodes, n)
	}
	return nodes
}
