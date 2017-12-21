package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_2_0_0"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/key"
)

// Files allows files to be injected into the master cloudconfig.
func (me *masterExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	cloudProviderConfFile, err := me.renderCloudProviderConfFile()
	if err != nil {
		return nil, microerror.Maskf(err, "renderCloudProviderConfFile")
	}

	getKeyVaultSecretsFile, err := me.renderGetKeyVaultSecretsFile()
	if err != nil {
		return nil, microerror.Maskf(err, "renderGetKeyVaultSecretsFile")
	}

	files := []k8scloudconfig.FileAsset{
		cloudProviderConfFile,
		getKeyVaultSecretsFile,
	}

	return files, nil
}

// Units allows systemd units to be injected into the master cloudconfig.
func (me *masterExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	getKeyVaultSecretsUnit, err := me.renderGetKeyVaultSecretsUnit()
	if err != nil {
		return nil, microerror.Maskf(err, "renderGetKeyVaultSecretsUnit")
	}

	units := []k8scloudconfig.UnitAsset{
		getKeyVaultSecretsUnit,
	}

	return units, nil
}

// VerbatimSections allows sections to be embedded in the master cloudconfig.
func (me *masterExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

func (me *masterExtension) renderCloudProviderConfFile() (k8scloudconfig.FileAsset, error) {
	params := newCloudProviderConfFileParams(me.AzureConfig, me.CustomObject)

	asset, err := renderCloudProviderConfFile(params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderGetKeyVaultSecretsFile() (k8scloudconfig.FileAsset, error) {
	params := getKeyVaultSecretsFileParams{
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
		params.Secrets = append(params.Secrets, s)
	}

	asset, err := renderGetKeyVaultSecretsFile(params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderGetKeyVaultSecretsUnit() (k8scloudconfig.UnitAsset, error) {
	params := me.CustomObject

	asset, err := renderGetKeyVaultSecretsUnit(params)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

// Files allows files to be injected into the master cloudconfig.
func (we *workerExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	cloudProviderConfFile, err := we.renderCloudProviderConfFile()
	if err != nil {
		return nil, microerror.Maskf(err, "renderCloudProviderConfFile")
	}

	getKeyVaultSecretsFile, err := we.renderGetKeyVaultSecretsFile()
	if err != nil {
		return nil, microerror.Maskf(err, "renderGetKeyVaultSecretsFile")
	}

	files := []k8scloudconfig.FileAsset{
		cloudProviderConfFile,
		getKeyVaultSecretsFile,
	}

	return files, nil
}

// Units allows systemd units to be injected into the master cloudconfig.
func (we *workerExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	getKeyVaultSecretsUnit, err := we.renderGetKeyVaultSecretsUnit()
	if err != nil {
		return nil, microerror.Maskf(err, "renderGetKeyVaultSecretsUnit")
	}

	units := []k8scloudconfig.UnitAsset{
		getKeyVaultSecretsUnit,
	}

	return units, nil
}

// VerbatimSections allows sections to be embedded in the worker cloudconfig.
func (we *workerExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

func (we *workerExtension) renderCloudProviderConfFile() (k8scloudconfig.FileAsset, error) {
	params := newCloudProviderConfFileParams(we.AzureConfig, we.CustomObject)

	asset, err := renderCloudProviderConfFile(params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (we *workerExtension) renderGetKeyVaultSecretsFile() (k8scloudconfig.FileAsset, error) {
	params := getKeyVaultSecretsFileParams{
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
		params.Secrets = append(params.Secrets, s)
	}

	asset, err := renderGetKeyVaultSecretsFile(params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (we *workerExtension) renderGetKeyVaultSecretsUnit() (k8scloudconfig.UnitAsset, error) {
	params := we.CustomObject

	asset, err := renderGetKeyVaultSecretsUnit(params)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func renderCloudProviderConfFile(params cloudProviderConfFileParams) (k8scloudconfig.FileAsset, error) {
	fileMeta := k8scloudconfig.FileMetadata{
		AssetContent: cloudProviderConfFileTemplate,
		Path:         cloudProviderConfFileName,
		Owner:        cloudProviderConfFileOwner,
		Permissions:  cloudProviderConfFilePermission,
	}

	content, err := k8scloudconfig.RenderAssetContent(fileMeta.AssetContent, params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	file := k8scloudconfig.FileAsset{
		Metadata: fileMeta,
		Content:  content,
	}

	return file, nil
}

func renderGetKeyVaultSecretsFile(params getKeyVaultSecretsFileParams) (k8scloudconfig.FileAsset, error) {
	fileMeta := k8scloudconfig.FileMetadata{
		AssetContent: getKeyVaultSecretsFileTemplate,
		Path:         getKeyVaultSecretsFileName,
		Owner:        getKeyVaultSecretsFileOwner,
		Permissions:  getKeyVaultSecretsFilePermission,
	}

	content, err := k8scloudconfig.RenderAssetContent(fileMeta.AssetContent, params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	file := k8scloudconfig.FileAsset{
		Metadata: fileMeta,
		Content:  content,
	}

	return file, nil
}

// TODO remove obj from parameters. This doesn't need params to render.
func renderGetKeyVaultSecretsUnit(obj providerv1alpha1.AzureConfig) (k8scloudconfig.UnitAsset, error) {
	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: getKeyVaultSecretsUnitTemplate,
		Name:         getKeyVaultSecretsUnitName,
		Enable:       true,
		Command:      "start",
	}

	content, err := k8scloudconfig.RenderAssetContent(unitMeta.AssetContent, obj)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	asset := k8scloudconfig.UnitAsset{
		Metadata: unitMeta,
		Content:  content,
	}

	return asset, nil
}
