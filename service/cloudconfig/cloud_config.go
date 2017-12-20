package cloudconfig

import (
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_2_0_0"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/key"
)

// Files allows files to be injected into the master cloudconfig.
func (me *masterExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	var newFiles []k8scloudconfig.FileAsset

	getSecretsScript, err := me.getMasterSecretsScript()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cloudProviderConf, err := me.masterCloudProviderConf()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	newFiles = []k8scloudconfig.FileAsset{
		getSecretsScript,
		cloudProviderConf,
	}

	return newFiles, nil
}

// Units allows systemd units to be injected into the master cloudconfig.
func (me *masterExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	unitsMeta := []k8scloudconfig.UnitMetadata{
		{
			AssetContent: getKeyVaultSecretsUnitTemplate,
			Name:         "get-keyvault-secrets.service",
			Enable:       true,
			Command:      "start",
		},
	}

	units, err := me.renderUnits(unitsMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return units, nil
}

// VerbatimSections allows sections to be embedded in the master cloudconfig.
func (me *masterExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

// masterCloudProviderConf returns Kubernetes cloud provider config for Azure.
func (me *masterExtension) masterCloudProviderConf() (k8scloudconfig.FileAsset, error) {
	// Prepare template parameters.
	params := struct {
		AzureCloudType    string
		Location          string
		ResourceGroup     string
		RouteTableName    string
		SecurityGroupName string
		SubnetName        string
		SubscriptionID    string
		TenantID          string
		VnetName          string
	}{
		AzureCloudType:    key.AzureCloudType(me.CustomObject),
		Location:          key.Location(me.CustomObject),
		ResourceGroup:     key.ResourceGroupName(me.CustomObject),
		RouteTableName:    key.RouteTableName(me.CustomObject),
		SecurityGroupName: key.MasterSecurityGroupName(me.CustomObject),
		SubnetName:        key.MasterSubnetName(me.CustomObject),
		SubscriptionID:    me.AzureConfig.SubscriptionID,
		TenantID:          me.AzureConfig.TenantID,
		VnetName:          key.VnetName(me.CustomObject),
	}

	// Prepare file metadata.
	meta := k8scloudconfig.FileMetadata{
		AssetContent: cloudProviderConfFileTemplate,
		Path:         cloudProviderConfFileName,
		Owner:        cloudProviderConfFileOwner,
		Permissions:  cloudProviderConfFilePermission,
	}

	// Generate template.
	content, err := k8scloudconfig.RenderAssetContent(meta.AssetContent, params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	asset := k8scloudconfig.FileAsset{
		Metadata: meta,
		Content:  content,
	}

	return asset, nil
}

// getMasterSecretsScript returns the script for downloading the master TLS
// certificates from Key Vault on startup.
func (me *masterExtension) getMasterSecretsScript() (k8scloudconfig.FileAsset, error) {
	secrets := keyVaultSecrets{
		VaultName: key.KeyVaultName(me.CustomObject),
		Secrets:   []keyVaultSecret{},
	}

	// Only file paths are needed here, so we don't care if certs.Cluster
	// is empty.
	for _, f := range certs.NewFilesClusterMaster(certs.Cluster{}) {
		s := keyVaultSecret{
			SecretName: key.KeyVaultKey(f.AbsolutePath),
			FileName:   f.AbsolutePath,
		}
		secrets.Secrets = append(secrets.Secrets, s)
	}

	getSecretsScript, err := me.renderGetSecretsScript(secrets)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return getSecretsScript, nil
}

// Files allows files to be injected into the worker cloudconfig.
func (we *workerExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	var newFiles []k8scloudconfig.FileAsset

	getSecretsScript, err := we.getWorkerSecretsScript()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cloudProviderConf, err := we.workerCloudProviderConf()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	newFiles = []k8scloudconfig.FileAsset{
		getSecretsScript,
		cloudProviderConf,
	}

	return newFiles, nil
}

// Units allows systemd units to be injected into the worker cloudconfig.
func (we *workerExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	unitsMeta := []k8scloudconfig.UnitMetadata{
		{
			AssetContent: getKeyVaultSecretsUnitTemplate,
			Name:         getKeyVaultSecretsUnitName,
			Enable:       true,
			Command:      "start",
		},
	}

	units, err := we.renderUnits(unitsMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return units, nil
}

// VerbatimSections allows sections to be embedded in the worker cloudconfig.
func (we *workerExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

// getWorkerSecretsScript returns the script for downloading the worker TLS
// certificates from Key Vault on startup.
func (we *workerExtension) getWorkerSecretsScript() (k8scloudconfig.FileAsset, error) {
	secrets := keyVaultSecrets{
		VaultName: key.KeyVaultName(we.CustomObject),
		Secrets:   []keyVaultSecret{},
	}

	// Only file paths are needed here, so we don't care if certs.Cluster
	// is empty.
	for _, f := range certs.NewFilesClusterWorker(certs.Cluster{}) {
		s := keyVaultSecret{
			SecretName: key.KeyVaultKey(f.AbsolutePath),
			FileName:   f.AbsolutePath,
		}
		secrets.Secrets = append(secrets.Secrets, s)
	}

	getSecretsScript, err := we.renderGetSecretsScript(secrets)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return getSecretsScript, nil
}

// workerCloudProviderConf returns Kubernetes cloud provider config for Azure.
func (we *workerExtension) workerCloudProviderConf() (k8scloudconfig.FileAsset, error) {
	// Prepare template parameters.
	params := struct {
		AzureCloudType    string
		Location          string
		ResourceGroup     string
		RouteTableName    string
		SecurityGroupName string
		SubnetName        string
		SubscriptionID    string
		TenantID          string
		VnetName          string
	}{
		AzureCloudType:    key.AzureCloudType(we.CustomObject),
		Location:          key.Location(we.CustomObject),
		ResourceGroup:     key.ResourceGroupName(we.CustomObject),
		RouteTableName:    key.RouteTableName(we.CustomObject),
		SecurityGroupName: key.WorkerSecurityGroupName(we.CustomObject),
		SubnetName:        key.WorkerSubnetName(we.CustomObject),
		SubscriptionID:    we.AzureConfig.SubscriptionID,
		TenantID:          we.AzureConfig.TenantID,
		VnetName:          key.VnetName(we.CustomObject),
	}

	// Prepare file metadata.
	meta := k8scloudconfig.FileMetadata{
		AssetContent: cloudProviderConfFileTemplate,
		Path:         cloudProviderConfFileName,
		Owner:        cloudProviderConfFileOwner,
		Permissions:  cloudProviderConfFilePermission,
	}

	// Generate template.
	content, err := k8scloudconfig.RenderAssetContent(meta.AssetContent, params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	asset := k8scloudconfig.FileAsset{
		Metadata: meta,
		Content:  content,
	}

	return asset, nil
}

func (c *cloudConfigExtension) renderGetSecretsScript(secrets keyVaultSecrets) (k8scloudconfig.FileAsset, error) {
	secretsMeta := k8scloudconfig.FileMetadata{
		AssetContent: getKeyVaultSecretsFileTemplate,
		Path:         getKeyVaultSecretsFileName,
		Owner:        getKeyVaultSecretsFileOwner,
		Permissions:  getKeyVaultSecretsFilePermission,
	}

	content, err := k8scloudconfig.RenderAssetContent(secretsMeta.AssetContent, secrets)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	downloadSecrets := k8scloudconfig.FileAsset{
		Metadata: secretsMeta,
		Content:  content,
	}

	return downloadSecrets, nil
}

func (c *cloudConfigExtension) renderUnits(unitsMeta []k8scloudconfig.UnitMetadata) ([]k8scloudconfig.UnitAsset, error) {
	units := make([]k8scloudconfig.UnitAsset, 0, len(unitsMeta))

	for _, unitMeta := range unitsMeta {
		content, err := k8scloudconfig.RenderAssetContent(unitMeta.AssetContent, c.CustomObject)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		unitAsset := k8scloudconfig.UnitAsset{
			Metadata: unitMeta,
			Content:  content,
		}

		units = append(units, unitAsset)
	}

	return units, nil
}
