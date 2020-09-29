package nodepool

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
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
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/vmsku"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodepool/template"
)

func (r Resource) getDesiredDeployment(ctx context.Context, storageAccountsClient *storage.AccountsClient, release *releasev1alpha1.Release, machinePool *capiexpv1alpha3.MachinePool, azureMachinePool *capzexpv1alpha3.AzureMachinePool, cluster *capiv1alpha3.Cluster, azureCluster *capzv1alpha3.AzureCluster) (azureresource.Deployment, error) {
	encrypterObject, err := r.getEncrypterObject(ctx, key.CertificateEncryptionSecretName(azureCluster))
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	storageAccountName := strings.Replace(fmt.Sprintf("%s%s", "gssa", azureCluster.GetName()), "-", "", -1)
	workerCloudConfig, err := r.getWorkerCloudConfig(ctx, storageAccountsClient, azureCluster.GetName(), storageAccountName, key.BlobContainerName(), key.BootstrapBlobName(*azureMachinePool), encrypterObject)
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

	sshPublicKey, err := base64.StdEncoding.DecodeString(azureMachinePool.Spec.Template.SSHPublicKey)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	currentReplicas := key.NodePoolMinReplicas(machinePool)
	if key.NodePoolMinReplicas(machinePool) != key.NodePoolMaxReplicas(machinePool) {
		// Autoscaler is enabled, will need to get the current number of replicas from the VMSS.
		candidate, err := r.getVMSScurrentScaling(ctx, cluster, azureCluster.GetName(), key.NodePoolVMSSName(azureMachinePool))
		if err != nil {
			return azureresource.Deployment{}, microerror.Mask(err)
		}

		// Function getVMSScurrentScaling returns 0 when the VMSS is not found.
		if candidate != 0 {
			currentReplicas = candidate
		}
	}

	var enableAcceleratedNetworking bool
	{
		if azureMachinePool.Spec.Template.AcceleratedNetworking != nil {
			enableAcceleratedNetworking = *azureMachinePool.Spec.Template.AcceleratedNetworking
		} else {
			// Enable accelerated networking if VM type supports it.
			client, err := r.ClientFactory.GetResourceSkusClient(ctx, cluster.ObjectMeta)
			if err != nil {
				return azureresource.Deployment{}, microerror.Mask(err)
			}

			vmSKU, err := vmsku.New(ctx, client, azureMachinePool.Spec.Template.VMSize)
			if err != nil {
				return azureresource.Deployment{}, microerror.Mask(err)
			}

			enableAcceleratedNetworking = vmSKU.HasCapability(vmsku.CapabilityAcceleratedNetworking)
		}
	}

	templateParameters := template.Parameters{
		AzureOperatorVersion:        project.Version(),
		ClusterID:                   azureCluster.GetName(),
		DataDisks:                   azureMachinePool.Spec.Template.DataDisks,
		EnableAcceleratedNetworking: enableAcceleratedNetworking,
		NodepoolName:                key.NodePoolVMSSName(azureMachinePool),
		OSImage: template.OSImage{
			Publisher: "kinvolk",
			Offer:     "flatcar-container-linux-free",
			SKU:       "stable",
			Version:   distroVersion,
		},
		Scaling: template.Scaling{
			MinReplicas:     key.NodePoolMinReplicas(machinePool),
			MaxReplicas:     key.NodePoolMaxReplicas(machinePool),
			CurrentReplicas: currentReplicas,
		},
		SSHPublicKey: string(sshPublicKey),
		SubnetName:   subnetName,
		VMCustomData: workerCloudConfig,
		VMSize:       azureMachinePool.Spec.Template.VMSize,
		VnetName:     vnetName,
		Zones:        machinePool.Spec.FailureDomains,
	}

	deployment, err := template.NewDeployment(templateParameters)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	return deployment, nil
}

func (r Resource) getSubnetName(azureMachinePool *capzexpv1alpha3.AzureMachinePool, azureCluster *capzv1alpha3.AzureCluster) (string, string, error) {
	for _, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		if azureMachinePool.Name == subnet.Name {
			if subnet.ID == "" {
				return "", "", microerror.Maskf(subnetNotReadyError, fmt.Sprintf("Subnet %#q ID field is empty, which means the Subnet is not Ready", subnet.Name))
			}

			return azureCluster.Spec.NetworkSpec.Vnet.Name, subnet.Name, nil
		}
	}

	return "", "", microerror.Maskf(notFoundError, "there is no allocated subnet for nodepool %#q in virtual network called %#q", azureMachinePool.Name, azureCluster.Spec.NetworkSpec.Vnet.ID)
}

func (r *Resource) getVMSScurrentScaling(ctx context.Context, cluster *capiv1alpha3.Cluster, resourceGroupName string, vmssName string) (int32, error) {
	client, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, cluster.ObjectMeta)
	if err != nil {
		return -1, microerror.Mask(err)
	}

	npVMSS, err := client.Get(ctx, resourceGroupName, vmssName)
	if IsNotFound(err) {
		// VMSS not found, scaling is unknown.
		return 0, nil
	} else if err != nil {
		return -1, microerror.Mask(err)
	}

	capacity64 := *npVMSS.Sku.Capacity

	// Unsafe type casting in theory, but in practice the capacity will never reach numbers not even close to 2^32.
	return int32(capacity64), nil
}

func (r *Resource) getWorkerCloudConfig(ctx context.Context, storageAccountsClient *storage.AccountsClient, resourceGroupName, storageAccountName, containerName, workerBlobName string, encrypterObject encrypter.Interface) (string, error) {
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

	sc, err := azblob.NewSharedKeyCredential(storageAccountName, primaryKey)
	if err != nil {
		return "", microerror.Mask(err)
	}

	p := azblob.NewPipeline(sc, azblob.PipelineOptions{})
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net", storageAccountName))
	serviceURL := azblob.NewServiceURL(*u, p)
	containerURL := serviceURL.NewContainerURL(containerName)

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
		Namespace: cluster.Namespace,
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
