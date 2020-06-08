package tcnp

import (
	"context"
	"fmt"
	"strings"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"k8s.io/kubernetes/pkg/apis/core"
	v1alpha32 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/helpers/vmss"
	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/encrypter"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	tcnp "github.com/giantswarm/azure-operator/v4/service/controller/resource/tcnp/template"
)

const (
	DeploymentTemplateChecksum   = "TemplateChecksum"
	DeploymentParametersChecksum = "ParametersChecksum"
	mainDeploymentName           = "tcnp"
)

// EnsureCreated will ensure the Deployment is created.
// It will create it if it doesn't exists, or it exists but it's out of date.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterID, exists := azureMachinePool.GetLabels()[label.Cluster]
	if !exists {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	machinePool := &v1alpha3.MachinePool{}
	err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePool.GetNamespace(), Name: azureMachinePool.GetName()}, machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	var desiredDeployment azureresource.Deployment
	var desiredDeploymentTemplateChk, desiredDeploymentParametersChk string
	{
		currentDeployment, err := deploymentsClient.Get(ctx, clusterID, mainDeploymentName)
		if IsNotFound(err) {
			desiredDeployment, err = r.newDeployment(ctx, *machinePool, azureMachinePool, map[string]interface{}{})
			if err != nil {
				return microerror.Mask(err)
			}

			desiredDeploymentTemplateChk, desiredDeploymentParametersChk, err = r.getDesiredDeploymentChecksums(ctx, desiredDeployment)
			if err != nil {
				return microerror.Mask(err)
			}
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			provisioningState := *currentDeployment.Properties.ProvisioningState

			r.debugger.LogFailedDeployment(ctx, currentDeployment, err)
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", provisioningState))

			if !key.IsFinalProvisioningState(provisioningState) {
				reconciliationcanceledcontext.SetCanceled(ctx)
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
				return nil
			}

			err = r.enrichControllerContext(ctx, deploymentsClient, azureMachinePool)
			if err != nil {
				return microerror.Mask(err)
			}

			currentDeploymentTemplateChk, currentDeploymentParametersChk, err := r.getCurrentDeploymentChecksums(ctx, azureMachinePool)
			if err != nil {
				return microerror.Mask(err)
			}

			desiredDeployment, err = r.newDeployment(ctx, *machinePool, azureMachinePool, map[string]interface{}{"initialProvisioning": "No"})
			if err != nil {
				return microerror.Mask(err)
			}

			desiredDeploymentTemplateChk, desiredDeploymentParametersChk, err = r.getDesiredDeploymentChecksums(ctx, desiredDeployment)
			if err != nil {
				return microerror.Mask(err)
			}

			if currentDeploymentIsUpToDate(currentDeploymentTemplateChk, desiredDeploymentTemplateChk, currentDeploymentParametersChk, desiredDeploymentParametersChk) {
				// No need to do anything else if deployment is up to date.
				r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")
				return nil
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")
		}
	}

	err = r.ensureDeployment(ctx, deploymentsClient, azureMachinePool.GetName(), desiredDeployment)
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

func (r *Resource) getDeploymentsClient(ctx context.Context) (*azureresource.DeploymentsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.DeploymentsClient, nil
}

func (r *Resource) getStorageAccountsClient(ctx context.Context) (*storage.AccountsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.StorageAccountsClient, nil
}

func currentDeploymentIsUpToDate(currentDeploymentTemplateChk, currentDeploymentParametersChk, desiredDeploymentTemplateChk, desiredDeploymentParametersChk string) bool {
	return currentDeploymentTemplateChk == desiredDeploymentTemplateChk && currentDeploymentParametersChk == desiredDeploymentParametersChk
}

func (r *Resource) saveDeploymentChecksumInStatus(ctx context.Context, customObject v1alpha32.AzureMachinePool, desiredDeploymentTemplateChk, desiredDeploymentParametersChk string) error {
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

func (r *Resource) ensureDeployment(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, resourceGroupName string, desiredDeployment azureresource.Deployment) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	res, err := deploymentsClient.CreateOrUpdate(ctx, resourceGroupName, mainDeploymentName, desiredDeployment)
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

func (r *Resource) enrichControllerContext(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, customObject v1alpha32.AzureMachinePool) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, deploymentsClient, customObject, "api_load_balancer_setup", "backendPoolId")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.APILBBackendPoolID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, deploymentsClient, customObject, "etcd_load_balancer_setup", "backendPoolId")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.EtcdLBBackendPoolID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, deploymentsClient, customObject, "virtual_network_setup", "masterSubnetID")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.MasterSubnetID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, deploymentsClient, customObject, "virtual_network_setup", "workerSubnetID")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.WorkerSubnetID = v
		}
	}

	return nil
}

func (r *Resource) getDeploymentOutputValue(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, customObject v1alpha32.AzureMachinePool, deploymentName string, outputName string) (string, error) {
	resourceGroupName := customObject.GetName()
	d, err := deploymentsClient.Get(ctx, resourceGroupName, deploymentName)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if d.Properties.Outputs == nil {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("cannot get output value '%s' of deployment '%s'", outputName, deploymentName))
		r.logger.LogCtx(ctx, "level", "warning", "message", "assuming deployment is in failed state")
		r.logger.LogCtx(ctx, "level", "warning", "message", "canceling controller context enrichment")
		return "", nil
	}

	m, err := key.ToMap(d.Properties.Outputs)
	if err != nil {
		return "", microerror.Mask(err)
	}
	v, ok := m[outputName]
	if !ok {
		return "", microerror.Maskf(missingOutputValueError, outputName)
	}
	m, err = key.ToMap(v)
	if err != nil {
		return "", microerror.Mask(err)
	}
	v, err = key.ToKeyValue(m)
	if err != nil {
		return "", microerror.Mask(err)
	}
	s, err := key.ToString(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return s, nil
}

func (r Resource) newDeployment(ctx context.Context, machinePool v1alpha3.MachinePool, azureMachinePool v1alpha32.AzureMachinePool, overwrites map[string]interface{}) (azureresource.Deployment, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	operatorVersion, exists := azureMachinePool.GetLabels()[label.OperatorVersion]
	if !exists {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	workerBlobName := key.WorkerBlobName(operatorVersion)
	err = r.checkCloudConfigBlob(ctx, workerBlobName)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	certificateEncryptionSecretName := fmt.Sprintf("%s-certificate-encryption", azureMachinePool.GetName())
	encrypterObject, err := r.getEncrypterObject(ctx, certificateEncryptionSecretName)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	storageAccountName := strings.Replace(fmt.Sprintf("%s%s", "gssa", azureMachinePool.GetName()), "-", "", -1)
	workerCloudConfig, err := r.getWorkerCloudConfig(ctx, azureMachinePool.GetName(), storageAccountName, workerBlobName, encrypterObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	distroVersion, err := key.OSVersion(cc.Release.Release)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	zones, err := r.getFailureDomains(ctx, azureMachinePool)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	templateParams := map[string]interface{}{
		"apiLBBackendPoolID":   cc.APILBBackendPoolID,
		"azureOperatorVersion": project.Version(),
		"clusterID":            azureMachinePool.GetName(),
		"dockerVolumeSizeGB":   50,
		"etcdLBBackendPoolID":  cc.EtcdLBBackendPoolID,
		"enableMSI":            r.vmssMSIEnabled,
		"kubeletVolumeSizeGB":  100,
		"vmCustomData":         workerCloudConfig,
		"sshUser":              "capi",
		"sshPublicKey":         azureMachinePool.Spec.Template.SSHPublicKey,
		"osImagePublisher":     "kinvolk",
		"osImageOffer":         "flatcar-container-linux-free",
		"osImageSKU":           "stable",
		"osImageVersion":       distroVersion,
		"replicas":             machinePool.Spec.Replicas,
		"subnetID":             cc.WorkerSubnetID,
		"vmSize":               azureMachinePool.Spec.Template.VMSize,
		"zones":                zones,
	}

	armTemplate, err := tcnp.GetARMTemplate()
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	d := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Incremental,
			Parameters: key.ToParameters(templateParams, overwrites),
			Template:   armTemplate,
		},
	}

	return d, nil
}

func (r *Resource) getFailureDomains(ctx context.Context, customObject v1alpha32.AzureMachinePool) ([]string, error) {
	machinePool := &v1alpha3.MachinePool{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: customObject.GetNamespace(), Name: customObject.GetName()}, machinePool)
	if err != nil {
		return []string{}, microerror.Mask(err)
	}

	// TODO: check FailureDomains are present in AzureCluster object. Those are the valid ones.

	return []string{*machinePool.Spec.Template.Spec.FailureDomain}, nil
}

func (r *Resource) checkCloudConfigBlob(ctx context.Context, workerBlobName string) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	cloudConfigURLs := []string{
		workerBlobName,
	}

	for _, cloudConfigURL := range cloudConfigURLs {
		blobURL := cc.ContainerURL.NewBlockBlobURL(cloudConfigURL)
		_, err := blobURL.GetProperties(ctx, azblob.BlobAccessConditions{})
		// if blob is not ready - stop instance resource reconciliation
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (r *Resource) getWorkerCloudConfig(ctx context.Context, resourceGroupName, storageAccountName, workerBlobName string, encrypterObject encrypter.Interface) (string, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	encryptionKey := encrypterObject.GetEncryptionKey()
	initialVector := encrypterObject.GetInitialVector()

	storageAccountsClient, err := r.getStorageAccountsClient(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	keys, err := storageAccountsClient.ListKeys(ctx, resourceGroupName, storageAccountName, "")
	if err != nil {
		return "", microerror.Mask(err)
	}

	if len(*(keys.Keys)) == 0 {
		return "", microerror.Maskf(executionFailedError, "storage account key's list is empty")
	}
	primaryKey := *(((*keys.Keys)[0]).Value)
	containerName := key.BlobContainerName()

	// Workers cloudconfig
	workerBlobURL, err := blobclient.GetBlobURL(workerBlobName, containerName, storageAccountName, primaryKey, cc.ContainerURL)
	if err != nil {
		return "", microerror.Mask(err)
	}
	return vmss.RenderCloudConfig(workerBlobURL, encryptionKey, initialVector, key.PrefixWorker())
}

func (r *Resource) getEncrypterObject(ctx context.Context, secretName string) (encrypter.Interface, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "retrieving encryptionkey")

	secret := &core.Secret{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: key.CertificateEncryptionNamespace, Name: secretName}, secret)
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
