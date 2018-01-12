package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/key"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_3_0_0"
	"github.com/giantswarm/microerror"
)

type workerExtension struct {
	AzureConfig  client.AzureConfig
	CustomObject providerv1alpha1.AzureConfig
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

	// Unit to format docker disk.
	formatDockerUnit, err := we.renderDockerDiskFormatUnit()
	if err != nil {
		return nil, microerror.Maskf(err, "renderDockerDiskFormatUnit")
	}

	// Unit to mount docker disk.
	mountDockerUnit, err := we.renderDockerMountUnit()
	if err != nil {
		return nil, microerror.Maskf(err, "renderDockerMountUnit")
	}

	units := []k8scloudconfig.UnitAsset{
		getKeyVaultSecretsUnit,
		formatDockerUnit,
		mountDockerUnit,
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
		Secrets:   []getKeyVaultSecretsFileParamsSecret{},
	}

	// Only file paths are needed here, so we don't care if certs.Cluster
	// is empty.
	for _, f := range certs.NewFilesClusterWorker(certs.Cluster{}) {
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

func (we *workerExtension) renderGetKeyVaultSecretsUnit() (k8scloudconfig.UnitAsset, error) {
	asset, err := renderGetKeyVaultSecretsUnit()
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (we *workerExtension) renderDockerMountUnit() (k8scloudconfig.UnitAsset, error) {
	params := diskParams{
		DiskName: "sdc",
	}

	asset, err := renderDockerMountUnit(params)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (we *workerExtension) renderDockerDiskFormatUnit() (k8scloudconfig.UnitAsset, error) {
	params := diskParams{
		DiskName: "sdc",
	}

	asset, err := renderDockerDiskFormatUnit(params)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}
