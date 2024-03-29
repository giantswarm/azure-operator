package nodepool

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"
	releasev1alpha1 "github.com/giantswarm/release-operator/v4/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/service/controller/azuremachinepool/handler/nodepool/template"

	"github.com/giantswarm/azure-operator/v8/pkg/helpers/vmss"
	"github.com/giantswarm/azure-operator/v8/pkg/label"
	"github.com/giantswarm/azure-operator/v8/pkg/project"
	"github.com/giantswarm/azure-operator/v8/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v8/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v8/service/controller/internal/vmsku"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

func (r Resource) getDesiredDeployment(ctx context.Context, storageAccountsClient *storage.AccountsClient, release *releasev1alpha1.Release, machinePool *capiexp.MachinePool, azureMachinePool *capzexp.AzureMachinePool, azureCluster *capz.AzureCluster, vmss compute.VirtualMachineScaleSet) (azureresource.Deployment, error) {
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
	kubernetesVersion, err := key.KubernetesVersion(*release)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	vnetName, subnetName, err := r.getSubnetName(azureMachinePool, azureCluster)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	currentReplicas := key.NodePoolMinReplicas(machinePool)
	if key.NodePoolMinReplicas(machinePool) != key.NodePoolMaxReplicas(machinePool) {
		// Autoscaler is enabled. Will need to use the current number of replicas from the VMSS if it exists.
		if !vmss.IsHTTPStatus(404) {
			// Update VMSS desired number of nodes in case existing desired
			// minimum number of nodes is bigger than current desired number of
			// nodes by cluster auto-scaler. It doesn't automatically increase
			// the number of nodes to minimum level so it must be triggered via
			// API here.
			if int32(*vmss.Sku.Capacity) > currentReplicas {
				currentReplicas = int32(*vmss.Sku.Capacity)
			}
		}
	}

	var enableAcceleratedNetworking bool
	{
		if azureMachinePool.Spec.Template.AcceleratedNetworking != nil {
			// The flag is set, just use its value.
			enableAcceleratedNetworking = *azureMachinePool.Spec.Template.AcceleratedNetworking
		} else {
			// The flag is not set.
			if vmss.IsHTTPStatus(404) {
				// Scale set does not exist yet.
				// We want to enable accelerated networking only if VM type supports it.
				enableAcceleratedNetworking, err = r.vmsku.HasCapability(ctx, azureMachinePool.Spec.Template.VMSize, vmsku.CapabilityAcceleratedNetworking)
				if err != nil {
					return azureresource.Deployment{}, microerror.Mask(err)
				}
			} else {
				// VMSS already exists, we want to stick with what is the current situation.
				cfgs := vmss.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations
				if cfgs != nil && len(*cfgs) > 0 {
					enableAcceleratedNetworking = *(*cfgs)[0].EnableAcceleratedNetworking
				} else {
					return azureresource.Deployment{}, microerror.Mask(unexpectedUpstreamResponseError)
				}
			}
		}
	}

	templateParameters := template.Parameters{
		AzureOperatorVersion:        project.Version(),
		CGroupsVersion:              key.CGroupVersion(machinePool),
		ClusterID:                   azureCluster.GetName(),
		DataDisks:                   azureMachinePool.Spec.Template.DataDisks,
		EnableAcceleratedNetworking: enableAcceleratedNetworking,
		NodepoolName:                key.NodePoolVMSSName(azureMachinePool),
		KubernetesVersion:           kubernetesVersion,
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
		SpotInstanceConfig: template.SpotInstanceConfig{
			Enabled:  key.NodePoolSpotInstancesEnabled(azureMachinePool),
			MaxPrice: key.NodePoolSpotInstancesMaxPrice(azureMachinePool),
		},
		StorageAccountType: azureMachinePool.Spec.Template.OSDisk.ManagedDisk.StorageAccountType,
		SubnetName:         subnetName,
		VMCustomData:       workerCloudConfig,
		VMSize:             azureMachinePool.Spec.Template.VMSize,
		VnetName:           vnetName,
		Zones:              machinePool.Spec.FailureDomains,
	}

	deployment, err := template.NewDeployment(templateParameters)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	return deployment, nil
}

func (r Resource) getSubnetName(azureMachinePool *capzexp.AzureMachinePool, azureCluster *capz.AzureCluster) (string, string, error) {
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
	r.Logger.Debugf(ctx, "retrieving encryptionkey")

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
func (r *Resource) getMachinePoolByName(ctx context.Context, namespace, name string) (*capiexp.MachinePool, error) {
	machinePool := &capiexp.MachinePool{}
	objectKey := ctrlclient.ObjectKey{Name: name, Namespace: namespace}
	if err := r.CtrlClient.Get(ctx, objectKey, machinePool); err != nil {
		return nil, err
	}

	r.Logger = r.Logger.With("machinePool", machinePool.Name)

	return machinePool, nil
}

// getOwnerMachinePool returns the MachinePool object owning the current resource.
func (r *Resource) getOwnerMachinePool(ctx context.Context, obj metav1.ObjectMeta) (*capiexp.MachinePool, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion == capiexp.GroupVersion.String() {
			return r.getMachinePoolByName(ctx, obj.Namespace, ref.Name)
		}
	}

	return nil, nil
}

func (r *Resource) getAzureClusterFromCluster(ctx context.Context, cluster *capi.Cluster) (*capz.AzureCluster, error) {
	azureCluster := &capz.AzureCluster{}
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
