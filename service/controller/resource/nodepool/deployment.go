package nodepool

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/helpers/vmss"
	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	instance "github.com/giantswarm/azure-operator/v4/service/controller/resource/nodepool/template"
)

func (r Resource) newDeployment(ctx context.Context, storageAccountsClient *storage.AccountsClient, release *releasev1alpha1.Release, machinePool *capiexpv1alpha3.MachinePool, azureMachinePool *capzexpv1alpha3.AzureMachinePool, azureCluster *capzv1alpha3.AzureCluster) (azureresource.Deployment, error) {
	operatorVersion, exists := azureMachinePool.GetLabels()[label.OperatorVersion]
	if !exists {
		return azureresource.Deployment{}, microerror.Mask(missingOperatorVersionLabel)
	}

	certificateEncryptionSecretName := fmt.Sprintf("%s-certificate-encryption", azureCluster.GetName())
	encrypterObject, err := r.getEncrypterObject(ctx, certificateEncryptionSecretName)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	storageAccountName := strings.Replace(fmt.Sprintf("%s%s", "gssa", azureCluster.GetName()), "-", "", -1)
	workerCloudConfig, err := r.getWorkerCloudConfig(ctx, storageAccountsClient, azureCluster.GetName(), storageAccountName, key.WorkerBlobName(operatorVersion), encrypterObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	distroVersion, err := key.OSVersion(*release)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	vnetName, subnetName, err := r.getSubnetName(azureMachinePool, azureCluster)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	templateParams := map[string]interface{}{
		"machinePoolVersion":      strconv.FormatInt(machinePool.ObjectMeta.Generation, 10),
		"azureMachinePoolVersion": strconv.FormatInt(azureMachinePool.ObjectMeta.Generation, 10),
		"azureOperatorVersion":    project.Version(),
		"clusterID":               azureCluster.GetName(),
		"dockerVolumeSizeGB":      "50",
		"kubeletVolumeSizeGB":     "100",
		"nodepoolName":            key.NodePoolVMSSName(azureMachinePool),
		"sshPublicKey":            azureMachinePool.Spec.Template.SSHPublicKey,
		"osImagePublisher":        "kinvolk",                      // azureMachinePool.Spec.Template.Image.Marketplace.Publisher,
		"osImageOffer":            "flatcar-container-linux-free", // azureMachinePool.Spec.Template.Image.Marketplace.Offer,
		"osImageSKU":              "stable",                       // azureMachinePool.Spec.Template.Image.Marketplace.SKU,
		"osImageVersion":          distroVersion,                  // azureMachinePool.Spec.Template.Image.Marketplace.Version,
		"replicas":                machinePool.Spec.Replicas,
		"vnetName":                vnetName,
		"subnetName":              subnetName,
		"vmSize":                  azureMachinePool.Spec.Template.VMSize,
		"zones":                   machinePool.Spec.FailureDomains,
		// This should come from the bootstrap operator.
		"vmCustomData": workerCloudConfig,
	}

	armTemplate, err := instance.GetARMTemplate()
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(templateParams),
			Template:   armTemplate,
		},
	}

	return d, nil
}

func (r Resource) getSubnetName(azureMachinePool *capzexpv1alpha3.AzureMachinePool, azureCluster *capzv1alpha3.AzureCluster) (string, string, error) {
	for _, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		if azureMachinePool.Name == subnet.Name {
			return azureCluster.Spec.NetworkSpec.Vnet.Name, subnet.Name, nil
		}
	}

	return "", "", microerror.Maskf(notFoundError, "there is no allocated subnet for nodepool %#q in virtual network called %#q", azureMachinePool.Name, azureCluster.Spec.NetworkSpec.Vnet.ID)
}

func (r *Resource) getWorkerCloudConfig(ctx context.Context, storageAccountsClient *storage.AccountsClient, resourceGroupName, storageAccountName, workerBlobName string, encrypterObject encrypter.Interface) (string, error) {
	encryptionKey := encrypterObject.GetEncryptionKey()
	initialVector := encrypterObject.GetInitialVector()

	keys, err := storageAccountsClient.ListKeys(ctx, resourceGroupName, storageAccountName, "")
	if err != nil {
		var errorMessage string
		if IsNotFound(err) {
			errorMessage = fmt.Sprintf("storage account %q not found", storageAccountName)
		} else {
			errorMessage = fmt.Sprintf("error while getting storage account %q", storageAccountName)
		}

		r.Logger.LogCtx(ctx, "level", "warning", "message", errorMessage)
		return "", microerror.Mask(err)
	}

	if len(*(keys.Keys)) == 0 {
		return "", microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}
	primaryKey := *(((*keys.Keys)[0]).Value)
	containerName := key.BlobContainerName()

	sc, err := azblob.NewSharedKeyCredential(storageAccountName, primaryKey)
	if err != nil {
		return "", microerror.Mask(err)
	}

	p := azblob.NewPipeline(sc, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", storageAccountName))
	serviceURL := azblob.NewServiceURL(*u, p)
	containerURL := serviceURL.NewContainerURL(key.BlobContainerName())

	workerBlobURL, err := blobclient.GetBlobURL(workerBlobName, containerName, storageAccountName, primaryKey, &containerURL)
	if err != nil {
		return "", microerror.Mask(err)
	}
	return vmss.RenderCloudConfig(workerBlobURL, encryptionKey, initialVector, key.PrefixWorker())
}

func (r *Resource) getEncrypterObject(ctx context.Context, secretName string) (encrypter.Interface, error) {
	r.Logger.LogCtx(ctx, "level", "debug", "message", "retrieving encryptionkey")

	secret := &corev1.Secret{}
	err := r.CtrlClient.Get(ctx, ctrlclient.ObjectKey{Namespace: key.CertificateEncryptionNamespace, Name: secretName}, secret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var enc *encrypter.Encrypter
	{
		if _, ok := secret.Data[key.CertificateEncryptionKeyName]; !ok {
			return nil, microerror.Maskf(executionFailedError, "encryption key not found in secret %q", secret.Name)
		}
		if _, ok := secret.Data[key.CertificateEncryptionIVName]; !ok {
			return nil, microerror.Maskf(executionFailedError, "encryption iv not found in secret %q", secret.Name)
		}
		c := encrypter.Config{
			Key: secret.Data[key.CertificateEncryptionKeyName],
			IV:  secret.Data[key.CertificateEncryptionIVName],
		}

		enc, err = encrypter.New(c)
		if err != nil {
			return nil, microerror.Mask(err)

		}
	}

	return enc, nil
}

// getMachinePoolByName finds and return a MachinePool object using the specified params.
func (r *Resource) getMachinePoolByName(ctx context.Context, namespace, name string) (*capiexpv1alpha3.MachinePool, error) {
	machinePool := &capiexpv1alpha3.MachinePool{}
	objectKey := ctrlclient.ObjectKey{Name: name, Namespace: namespace}
	if err := r.CtrlClient.Get(ctx, objectKey, machinePool); err != nil {
		return nil, err
	}

	r.Logger = r.Logger.With("machinePool", machinePool.Name)

	return machinePool, nil
}

// getOwnerMachinePool returns the MachinePool object owning the current resource.
func (r *Resource) getOwnerMachinePool(ctx context.Context, obj metav1.ObjectMeta) (*capiexpv1alpha3.MachinePool, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion == capiexpv1alpha3.GroupVersion.String() {
			return r.getMachinePoolByName(ctx, obj.Namespace, ref.Name)
		}
	}

	return nil, nil
}

func (r *Resource) getAzureClusterFromCluster(ctx context.Context, cluster *capiv1alpha3.Cluster) (*capzv1alpha3.AzureCluster, error) {
	azureCluster := &capzv1alpha3.AzureCluster{}
	azureClusterName := ctrlclient.ObjectKey{
		Namespace: cluster.Spec.InfrastructureRef.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	err := r.CtrlClient.Get(ctx, azureClusterName, azureCluster)
	if err != nil {
		return azureCluster, microerror.Mask(err)
	}

	r.Logger = r.Logger.With("azureCluster", azureCluster.Name)

	return azureCluster, nil
}

func (r *Resource) getReleaseFromMetadata(ctx context.Context, obj metav1.ObjectMeta) (*releasev1alpha1.Release, error) {
	release := &releasev1alpha1.Release{}
	releaseVersion, exists := obj.GetLabels()[label.ReleaseVersion]
	if !exists {
		return release, microerror.Mask(missingReleaseVersionLabel)
	}
	if !strings.HasPrefix(releaseVersion, "v") {
		releaseVersion = fmt.Sprintf("v%s", releaseVersion)
	}

	err := r.CtrlClient.Get(ctx, ctrlclient.ObjectKey{Namespace: "", Name: releaseVersion}, release)
	if err != nil {
		return release, microerror.Mask(err)
	}

	r.Logger = r.Logger.With("release", release.Name)

	return release, nil
}
