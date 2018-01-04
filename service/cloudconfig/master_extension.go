package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/key"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_3_0_0"
	"github.com/giantswarm/microerror"
)

type masterExtension struct {
	AzureConfig  client.AzureConfig
	CustomObject providerv1alpha1.AzureConfig
}

// Files allows files to be injected into the master cloudconfig.
func (me *masterExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	calicoAzureFile, err := me.renderCalicoAzureFile()
	if err != nil {
		return nil, microerror.Maskf(err, "renderCalicoAzureFile")
	}

	cloudProviderConfFile, err := me.renderCloudProviderConfFile()
	if err != nil {
		return nil, microerror.Maskf(err, "renderCloudProviderConfFile")
	}

	defaultStorageClassFile, err := me.renderDefaultStorageClassFile()
	if err != nil {
		return nil, microerror.Maskf(err, "renderDefaultStorageClassFile")
	}

	getKeyVaultSecretsFile, err := me.renderGetKeyVaultSecretsFile()
	if err != nil {
		return nil, microerror.Maskf(err, "renderGetKeyVaultSecretsFile")
	}

	files := []k8scloudconfig.FileAsset{
		calicoAzureFile,
		cloudProviderConfFile,
		defaultStorageClassFile,
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

func (me *masterExtension) renderCalicoAzureFile() (k8scloudconfig.FileAsset, error) {
	params := newCalicoAzureFileParams(me.CustomObject)

	asset, err := renderCalicoAzureFile(params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderCloudProviderConfFile() (k8scloudconfig.FileAsset, error) {
	params := newCloudProviderConfFileParams(me.AzureConfig, me.CustomObject)

	asset, err := renderCloudProviderConfFile(params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderDefaultStorageClassFile() (k8scloudconfig.FileAsset, error) {
	asset, err := renderDefaultStorageClassFile()
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderGetKeyVaultSecretsFile() (k8scloudconfig.FileAsset, error) {
	params := getKeyVaultSecretsFileParams{
		VaultName: key.KeyVaultName(me.CustomObject),
		Secrets:   []getKeyVaultSecretsFileParamsSecret{},
	}

	// Only file paths are needed here, so we don't care if certs.Cluster
	// is empty.
	for _, f := range certs.NewFilesClusterMaster(certs.Cluster{}) {
		s := getKeyVaultSecretsFileParamsSecret{
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
	asset, err := renderGetKeyVaultSecretsUnit()
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}
