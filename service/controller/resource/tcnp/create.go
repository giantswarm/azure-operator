package tcnp

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/pkg/helpers/vmss"
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
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	location := "westeurope"

	var desiredDeployment azureresource.Deployment
	var desiredDeploymentTemplateChk, desiredDeploymentParametersChk string
	{
		currentDeployment, err := deploymentsClient.Get(ctx, key.ClusterID(cr), mainDeploymentName)
		if IsNotFound(err) {
			desiredDeployment, err = r.newDeployment(ctx, cr, map[string]interface{}{}, location)
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

			err = r.enrichControllerContext(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			currentDeploymentTemplateChk, currentDeploymentParametersChk, err := r.getCurrentDeploymentChecksums(ctx, cr)
			if err != nil {
				return microerror.Mask(err)
			}

			desiredDeployment, err = r.newDeployment(ctx, cr, map[string]interface{}{"initialProvisioning": "No"}, location)
			if err != nil {
				return microerror.Mask(err)
			}

			desiredDeploymentTemplateChk, desiredDeploymentParametersChk, err = r.getDesiredDeploymentChecksums(ctx, desiredDeployment)
			if err != nil {
				return microerror.Mask(err)
			}

			if currentTemplateIsUpToDate(currentDeploymentTemplateChk, desiredDeploymentTemplateChk, currentDeploymentParametersChk, desiredDeploymentParametersChk) {
				// No need to do anything else if deployment is up to date.
				r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")
				return nil
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")
		}
	}

	err = r.ensureDeployment(ctx, cr, desiredDeployment)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.saveDeploymentChecksumInStatus(ctx, cr, desiredDeploymentTemplateChk, desiredDeploymentParametersChk)
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

func (r *Resource) getCurrentDeploymentChecksums(ctx context.Context, cr providerv1alpha1.AzureConfig) (string, string, error) {
	currentDeploymentTemplateChk, err := r.getResourceStatus(cr, DeploymentTemplateChecksum)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	currentDeploymentParametersChk, err := r.getResourceStatus(cr, DeploymentParametersChecksum)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	return currentDeploymentTemplateChk, currentDeploymentParametersChk, nil
}

func (r *Resource) getDesiredDeploymentChecksums(ctx context.Context, desiredDeployment azureresource.Deployment) (string, string, error) {
	desiredDeploymentTemplateChk, err := getDeploymentTemplateChecksum(desiredDeployment)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	desiredDeploymentParametersChk, err := getDeploymentParametersChecksum(desiredDeployment)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	return desiredDeploymentTemplateChk, desiredDeploymentParametersChk, nil
}

func currentTemplateIsUpToDate(currentDeploymentTemplateChk, currentDeploymentParametersChk, desiredDeploymentTemplateChk, desiredDeploymentParametersChk string) bool {
	return currentDeploymentTemplateChk == desiredDeploymentTemplateChk && currentDeploymentParametersChk == desiredDeploymentParametersChk
}

func (r *Resource) saveDeploymentChecksumInStatus(ctx context.Context, cr providerv1alpha1.AzureConfig, desiredDeploymentTemplateChk, desiredDeploymentParametersChk string) error {
	var err error
	if desiredDeploymentTemplateChk != "" {
		err = r.setResourceStatus(cr, DeploymentTemplateChecksum, desiredDeploymentTemplateChk)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentTemplateChecksum, desiredDeploymentTemplateChk))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentTemplateChecksum))
	}

	if desiredDeploymentParametersChk != "" {
		err = r.setResourceStatus(cr, DeploymentParametersChecksum, desiredDeploymentParametersChk)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentParametersChecksum, desiredDeploymentParametersChk))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentParametersChecksum))
	}

	return nil
}

func (r *Resource) ensureDeployment(ctx context.Context, cr providerv1alpha1.AzureConfig, desiredDeployment azureresource.Deployment) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	res, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(cr), mainDeploymentName, desiredDeployment)
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

func (r *Resource) enrichControllerContext(ctx context.Context, customObject providerv1alpha1.AzureConfig) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "api_load_balancer_setup", "backendPoolId")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.APILBBackendPoolID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "etcd_load_balancer_setup", "backendPoolId")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.EtcdLBBackendPoolID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "virtual_network_setup", "masterSubnetID")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.MasterSubnetID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "virtual_network_setup", "workerSubnetID")
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

func (r *Resource) getDeploymentOutputValue(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentName string, outputName string) (string, error) {
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}
	d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), deploymentName)
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

func (r Resource) newDeployment(ctx context.Context, customObject providerv1alpha1.AzureConfig, overwrites map[string]interface{}, location string) (azureresource.Deployment, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	err = r.checkCloudConfigBlob(ctx, customObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	certificateEncryptionSecretName := key.CertificateEncryptionSecretName(customObject)
	encrypterObject, err := r.getEncrypterObject(ctx, certificateEncryptionSecretName)
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "encryptionkey secret is not found", "secretname", certificateEncryptionSecretName)
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return azureresource.Deployment{}, nil
	} else if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	workerCloudConfig, err := r.getWorkerCloudConfig(ctx, customObject, encrypterObject)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	distroVersion, err := key.OSVersion(cc.Release.Release)
	if err != nil {
		return azureresource.Deployment{}, microerror.Mask(err)
	}

	templateParams := map[string]interface{}{
		"apiLBBackendPoolID":    cc.APILBBackendPoolID,
		"azureOperatorVersion":  project.Version(),
		"clusterID":             key.ClusterID(customObject),
		"etcdLBBackendPoolID":   cc.EtcdLBBackendPoolID,
		"vmssMSIEnabled":        r.azure.MSI.Enabled,
		"workerCloudConfigData": workerCloudConfig,
		"workerNodes":           vmss.GetWorkerNodesConfiguration(customObject, distroVersion),
		"workerSubnetID":        cc.WorkerSubnetID,
		"zones":                 key.AvailabilityZones(customObject, location),
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

func (r *Resource) checkCloudConfigBlob(ctx context.Context, customObject providerv1alpha1.AzureConfig) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	prefixWorker := key.PrefixWorker()
	workerBlobName := key.BlobName(customObject, prefixWorker)
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

func (r *Resource) getWorkerCloudConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, encrypterObject encrypter.Interface) (string, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	prefixWorker := key.PrefixWorker()
	workerBlobName := key.BlobName(customObject, prefixWorker)
	encryptionKey := encrypterObject.GetEncryptionKey()
	initialVector := encrypterObject.GetInitialVector()

	storageAccountsClient, err := r.getStorageAccountsClient(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	groupName := key.ResourceGroupName(customObject)
	storageAccountName := key.StorageAccountName(customObject)
	keys, err := storageAccountsClient.ListKeys(ctx, groupName, storageAccountName, "")
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
	return vmss.RenderCloudConfig(workerBlobURL, encryptionKey, initialVector, prefixWorker)
}

func (r *Resource) getEncrypterObject(ctx context.Context, secretName string) (encrypter.Interface, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "retrieving encryptionkey")

	secret, err := r.k8sClient.CoreV1().Secrets(key.CertificateEncryptionNamespace).Get(secretName, metav1.GetOptions{})
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
