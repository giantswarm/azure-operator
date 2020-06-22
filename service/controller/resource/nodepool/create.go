package nodepool

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-storage-blob-go/azblob"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/annotation"
	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/pkg/helpers/vmss"
	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	nodepool "github.com/giantswarm/azure-operator/v4/service/controller/resource/nodepool/template"
)

const (
	DeploymentTemplateChecksum   = "TemplateChecksum"
	DeploymentParametersChecksum = "ParametersChecksum"
	mainDeploymentName           = "nodepool"
)

// EnsureCreated will ensure the Deployment is created.
// It will create it if it doesn't exists, or it exists but it's out of date.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.ctrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	azureCluster, err := r.getAzureClusterFromCluster(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	release, err := r.getReleaseFromMetadata(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	tenantClusterAzureClientSet, err := r.getTenantClusterAzureClientSet(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterID, exists := azureMachinePool.GetLabels()[label.Cluster]
	if !exists {
		return microerror.Mask(missingClusterLabel)
	}

	currentDeployment, err := tenantClusterAzureClientSet.DeploymentsClient.Get(ctx, clusterID, fmt.Sprintf("%s-%s", mainDeploymentName, azureMachinePool.Name))
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ARM deployment does not exist yet")
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		provisioningState := *currentDeployment.Properties.ProvisioningState
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", provisioningState))

		if !key.IsFinalProvisioningState(provisioningState) {
			reconciliationcanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			return nil
		}
	}

	desiredDeployment, err := r.newDeployment(ctx, tenantClusterAzureClientSet, release, *machinePool, azureMachinePool, azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	desiredDeploymentTemplateChk, desiredDeploymentParametersChk, err := r.getDesiredDeploymentChecksums(ctx, desiredDeployment)
	if err != nil {
		return microerror.Mask(err)
	}

	currentDeploymentTemplateChk, currentDeploymentParametersChk, err := r.getCurrentDeploymentChecksums(ctx, azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	if currentDeploymentIsUpToDate(currentDeploymentTemplateChk, desiredDeploymentTemplateChk, currentDeploymentParametersChk, desiredDeploymentParametersChk) {
		// No need to do anything else if deployment is up to date.
		r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")

	err = r.ensureDeployment(ctx, azureMachinePool, tenantClusterAzureClientSet.DeploymentsClient, azureCluster.GetName(), desiredDeployment)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.saveDeploymentChecksumInStatus(ctx, azureMachinePool, desiredDeploymentTemplateChk, desiredDeploymentParametersChk)
	if err != nil {
		return microerror.Mask(err)
	}

	// We just send request to create the deployment. It will take a while, let's cancel and check later.
	r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
	reconciliationcanceledcontext.SetCanceled(ctx)

	return nil
}

func currentDeploymentIsUpToDate(currentDeploymentTemplateChk, currentDeploymentParametersChk, desiredDeploymentTemplateChk, desiredDeploymentParametersChk string) bool {
	return currentDeploymentTemplateChk == desiredDeploymentTemplateChk && currentDeploymentParametersChk == desiredDeploymentParametersChk
}

func (r *Resource) saveDeploymentChecksumInStatus(ctx context.Context, customObject capzexpv1alpha3.AzureMachinePool, desiredDeploymentTemplateChk, desiredDeploymentParametersChk string) error {
	var err error
	if desiredDeploymentTemplateChk != "" {
		err = r.setResourceStatus(ctx, customObject, DeploymentTemplateChecksum, desiredDeploymentTemplateChk)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentTemplateChecksum, desiredDeploymentTemplateChk))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentTemplateChecksum))
	}

	if desiredDeploymentParametersChk != "" {
		err = r.setResourceStatus(ctx, customObject, DeploymentParametersChecksum, desiredDeploymentParametersChk)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentParametersChecksum, desiredDeploymentParametersChk))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentParametersChecksum))
	}

	return nil
}

func (r *Resource) ensureDeployment(ctx context.Context, azureMachinePool capzexpv1alpha3.AzureMachinePool, deploymentsClient *azureresource.DeploymentsClient, resourceGroupName string, desiredDeployment azureresource.Deployment) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	res, err := deploymentsClient.CreateOrUpdate(ctx, resourceGroupName, fmt.Sprintf("%s-%s", mainDeploymentName, azureMachinePool.Name), desiredDeployment)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", desiredDeployment), "stack", microerror.JSON(microerror.Mask(err)))

		return microerror.Mask(err)
	}

	deploymentExtended, err := deploymentsClient.CreateOrUpdateResponder(res.Response())
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", deploymentExtended), "stack", microerror.JSON(microerror.Mask(err)))

		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment")

	return nil
}

func (r Resource) newDeployment(ctx context.Context, azureClientSet *client.AzureClientSet, release releasev1alpha1.Release, machinePool capiexpv1alpha3.MachinePool, azureMachinePool capzexpv1alpha3.AzureMachinePool, azureCluster capzv1alpha3.AzureCluster) (azureresource.Deployment, error) {
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
	workerCloudConfig, err := r.getWorkerCloudConfig(ctx, azureClientSet, azureCluster.GetName(), storageAccountName, key.WorkerBlobName(operatorVersion), encrypterObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	distroVersion, err := key.OSVersion(release)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	zones, err := r.getFailureDomains(ctx, azureCluster, machinePool)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	subnetID, err := r.getSubnetID(ctx, azureClientSet, azureMachinePool, azureCluster)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	templateParams := map[string]interface{}{
		"azureOperatorVersion": project.Version(),
		"clusterID":            azureCluster.GetName(),
		"dockerVolumeSizeGB":   "50",
		"kubeletVolumeSizeGB":  "100",
		"sshPublicKey":         azureMachinePool.Spec.Template.SSHPublicKey,
		"osImagePublisher":     "kinvolk",                      // azureMachinePool.Spec.Template.Image.Marketplace.Publisher,
		"osImageOffer":         "flatcar-container-linux-free", // azureMachinePool.Spec.Template.Image.Marketplace.Offer,
		"osImageSKU":           "stable",                       // azureMachinePool.Spec.Template.Image.Marketplace.SKU,
		"osImageVersion":       distroVersion,                  // azureMachinePool.Spec.Template.Image.Marketplace.Version,
		"replicas":             machinePool.Spec.Replicas,
		"subnetID":             subnetID,
		// This should come from the bootstrap operator.
		"vmCustomData": workerCloudConfig,
		"vmSize":       azureMachinePool.Spec.Template.VMSize,
		"zones":        zones,
	}

	armTemplate, err := nodepool.GetARMTemplate()
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

func (r Resource) getSubnetID(ctx context.Context, azureClientSet *client.AzureClientSet, azureMachinePool capzexpv1alpha3.AzureMachinePool, azureCluster capzv1alpha3.AzureCluster) (string, error) {
	subnetCIDR, exists := azureMachinePool.GetAnnotations()[annotation.AzureMachinePoolSubnet]
	if !exists {
		return "", microerror.Mask(missingSubnetLabel)
	}

	for _, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		if subnetCIDR == subnet.CidrBlock {
			return subnet.ID, nil
		}
	}

	//subnetsInVnet, err := azureClientSet.SubnetsClient.List(ctx, azureCluster.GetName(), azureCluster.Spec.NetworkSpec.Vnet.ID)
	//if err != nil {
	//	return "", microerror.Mask(err)
	//}
	//
	//for _, subnet := range subnetsInVnet.Values() {
	//	r.logger.LogCtx(ctx, "message", "Comparing CIDR to subnet address prefix", "cidr", subnetCIDR, "addressPrefix", *subnet.AddressPrefix)
	//	if *subnet.AddressPrefix == subnetCIDR {
	//		return *subnet.ID, nil
	//	}
	//}

	return "", microerror.Maskf(notFoundError, "subnet with CIDR %#q was not found in virtual network called %#q", subnetCIDR, azureCluster.Spec.NetworkSpec.Vnet.ID)
}

func (r *Resource) getFailureDomains(ctx context.Context, azureCluster capzv1alpha3.AzureCluster, machinePool capiexpv1alpha3.MachinePool) ([]string, error) {
	var validFailureDomain bool
	allowedFailureDomains := azureCluster.Status.FailureDomains.GetIDs()
	for _, id := range allowedFailureDomains {
		if id == machinePool.Spec.Template.Spec.FailureDomain {
			validFailureDomain = true
		}
	}

	if !validFailureDomain {
		return []string{}, microerror.Maskf(notAvailableFailureDomain, "nodepool availability zone %#q is not available in the region", *machinePool.Spec.Template.Spec.FailureDomain)
	}

	return []string{*machinePool.Spec.Template.Spec.FailureDomain}, nil
}

func (r *Resource) getWorkerCloudConfig(ctx context.Context, azureClientSet *client.AzureClientSet, resourceGroupName, storageAccountName, workerBlobName string, encrypterObject encrypter.Interface) (string, error) {
	encryptionKey := encrypterObject.GetEncryptionKey()
	initialVector := encrypterObject.GetInitialVector()

	keys, err := azureClientSet.StorageAccountsClient.ListKeys(ctx, resourceGroupName, storageAccountName, "")
	if err != nil {
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
	r.logger.LogCtx(ctx, "level", "debug", "message", "retrieving encryptionkey")

	secret := &corev1.Secret{}
	err := r.ctrlClient.Get(ctx, ctrlclient.ObjectKey{Namespace: key.CertificateEncryptionNamespace, Name: secretName}, secret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var enc *encrypter.Encrypter
	{
		if _, ok := secret.Data[key.CertificateEncryptionKeyName]; !ok {
			return nil, microerror.Maskf(invalidConfigError, "encryption key not found in secret %q", secret.Name)
		}
		if _, ok := secret.Data[key.CertificateEncryptionIVName]; !ok {
			return nil, microerror.Maskf(invalidConfigError, "encryption iv not found in secret %q", secret.Name)
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
	if err := r.ctrlClient.Get(ctx, objectKey, machinePool); err != nil {
		return nil, err
	}

	r.logger = r.logger.With("machinePool", machinePool.Name)

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

func (r *Resource) getAzureClusterFromCluster(ctx context.Context, cluster *capiv1alpha3.Cluster) (capzv1alpha3.AzureCluster, error) {
	azureCluster := capzv1alpha3.AzureCluster{}
	azureClusterName := ctrlclient.ObjectKey{
		Namespace: cluster.Spec.InfrastructureRef.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	err := r.ctrlClient.Get(ctx, azureClusterName, &azureCluster)
	if err != nil {
		return azureCluster, microerror.Mask(err)
	}

	r.logger = r.logger.With("azureCluster", azureCluster.Name)

	return azureCluster, nil
}

func (r *Resource) getReleaseFromMetadata(ctx context.Context, obj metav1.ObjectMeta) (releasev1alpha1.Release, error) {
	release := releasev1alpha1.Release{}
	releaseVersion, exists := obj.GetLabels()[label.ReleaseVersion]
	if !exists {
		return release, microerror.Mask(missingReleaseVersionLabel)
	}
	if !strings.HasPrefix(releaseVersion, "v") {
		releaseVersion = fmt.Sprintf("v%s", releaseVersion)
	}

	err := r.ctrlClient.Get(ctx, ctrlclient.ObjectKey{Namespace: "", Name: releaseVersion}, &release)
	if err != nil {
		return release, microerror.Mask(err)
	}

	r.logger = r.logger.With("release", release.Name)

	return release, nil
}

func (r *Resource) getTenantClusterAzureClientSet(ctx context.Context, cluster *capiv1alpha3.Cluster) (*client.AzureClientSet, error) {
	credentialSecret, err := r.getCredentialSecret(ctx, *cluster)
	if err != nil {
		return &client.AzureClientSet{}, microerror.Mask(err)
	}

	organizationAzureClientCredentialsConfig, subscriptionID, partnerID, err := credential.GetOrganizationAzureCredentialsFromCredentialSecret(ctx, r.ctrlClient, *credentialSecret, r.gsClientCredentialsConfig.TenantID)
	if err != nil {
		return &client.AzureClientSet{}, microerror.Mask(err)
	}

	return client.NewAzureClientSet(organizationAzureClientCredentialsConfig, subscriptionID, partnerID)
}
