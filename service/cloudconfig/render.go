package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_2_0_0"
	"github.com/giantswarm/microerror"
)

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
