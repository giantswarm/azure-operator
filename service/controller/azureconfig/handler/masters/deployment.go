package masters

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-storage-blob-go/azblob"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers/vmss"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/masters/template"
	"github.com/giantswarm/azure-operator/v5/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/vmsku"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
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

	masterBlobName := key.BlobName(&obj, prefixMaster)
	cloudConfigURLs := []string{
		masterBlobName,
	}

	distroVersion, err := key.OSVersion(cc.Release.Release)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	for _, k := range cloudConfigURLs {
		blobURL := cc.ContainerURL.NewBlockBlobURL(k)
		_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
		// if blob is not ready - stop instance resource reconciliation
		if err != nil {
			return azureresource.Deployment{}, microerror.Mask(err)
		}
	}

	certificateEncryptionSecretName := key.CertificateEncryptionSecretName(&obj)
	encrypter, err := r.GetEncrypterObject(ctx, certificateEncryptionSecretName)
	if apierrors.IsNotFound(microerror.Cause(err)) {
		r.Logger.Debugf(ctx, "encryptionkey secret is not found", "secretname", certificateEncryptionSecretName)
		resourcecanceledcontext.SetCanceled(ctx)
		r.Logger.Debugf(ctx, "canceling resource")
		return azureresource.Deployment{}, nil
	} else if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	encryptionKey := encrypter.GetEncryptionKey()
	initialVector := encrypter.GetInitialVector()

	storageAccountsClient, err := r.ClientFactory.GetStorageAccountsClient(ctx, obj.ObjectMeta)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	groupName := key.ResourceGroupName(obj)
	storageAccountName := key.StorageAccountName(&obj)
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
	masterCloudConfig, err := vmss.RenderCloudConfig(masterBlobURL, encryptionKey, initialVector, prefixMaster)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	masterNodes := vmss.GetMasterNodesConfiguration(obj, distroVersion)

	var storageAccountType string
	{
		premium, err := r.vmSku.HasCapability(ctx, masterNodes[0].VMSize, vmsku.CapabilityPremiumIO)
		if err != nil {
			return azureresource.Deployment{}, microerror.Mask(err)
		}

		if premium {
			storageAccountType = string(compute.StorageAccountTypesPremiumLRS)
		} else {
			storageAccountType = string(compute.StorageAccountTypeStandardLRS)
		}
	}

	defaultParams := map[string]interface{}{
		"masterLBBackendPoolID": cc.MasterLBBackendPoolID,
		"azureOperatorVersion":  project.Version(),
		"clusterID":             key.ClusterID(&obj),
		"masterCloudConfigData": masterCloudConfig,
		"masterNodes":           masterNodes,
		"masterSubnetID":        cc.MasterSubnetID,
		"storageAccountType":    storageAccountType,
		"vmssMSIEnabled":        r.Azure.MSI.Enabled,
		"zones":                 key.AvailabilityZones(obj, location),
	}

	armTemplate, err := template.GetARMTemplate()
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(defaultParams, overwrites),
			Template:   armTemplate,
		},
	}

	return d, nil
}
