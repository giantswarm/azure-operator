package cloudconfig

import (
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_3_0_0"
	"github.com/giantswarm/microerror"
)

func renderCalicoAzureFile(params calicoAzureFileParams) (k8scloudconfig.FileAsset, error) {
	fileMeta := k8scloudconfig.FileMetadata{
		AssetContent: calicoAzureFileTemplate,
		Path:         calicoAzureFileName,
		Owner:        calicoAzureFileOwner,
		Permissions:  calicoAzureFilePermission,
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

func renderDefaultStorageClassFile() (k8scloudconfig.FileAsset, error) {
	params := struct{}{}

	fileMeta := k8scloudconfig.FileMetadata{
		AssetContent: defaultStorageClassFileTemplate,
		Path:         defaultStorageClassFileName,
		Owner:        defaultStorageClassFileOwner,
		Permissions:  defaultStorageClassFilePermission,
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

func renderGetKeyVaultSecretsUnit() (k8scloudconfig.UnitAsset, error) {
	params := struct{}{}

	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: getKeyVaultSecretsUnitTemplate,
		Name:         getKeyVaultSecretsUnitName,
		Enable:       true,
		Command:      "start",
	}

	content, err := k8scloudconfig.RenderAssetContent(unitMeta.AssetContent, params)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	asset := k8scloudconfig.UnitAsset{
		Metadata: unitMeta,
		Content:  content,
	}

	return asset, nil
}
