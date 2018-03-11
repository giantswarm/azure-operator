package cloudconfig

import (
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_3_2_2"
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

func renderCertificatesFiles(certFiles certs.Files) ([]k8scloudconfig.FileAsset, error) {
	params := struct{}{}

	var fileMetas []k8scloudconfig.FileMetadata
	for _, f := range certFiles {
		m := k8scloudconfig.FileMetadata{
			AssetContent: string(f.Data),
			Path:         f.AbsolutePath,
			Owner:        certFileOwner,
			Permissions:  certFilePermission,
		}
		fileMetas = append(fileMetas, m)
	}

	var files []k8scloudconfig.FileAsset
	for _, m := range fileMetas {
		content, err := k8scloudconfig.RenderAssetContent(m.AssetContent, params)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		f := k8scloudconfig.FileAsset{
			Metadata: m,
			Content:  content,
		}
		files = append(files, f)
	}

	return files, nil
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

func renderEtcdMountUnit(params diskParams) (k8scloudconfig.UnitAsset, error) {
	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: etcdMountUnitTemplate,
		Name:         etcdMountUnitName,
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

func renderEtcdDiskFormatUnit(params diskParams) (k8scloudconfig.UnitAsset, error) {
	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: etcdDiskFormatUnitTemplate,
		Name:         etcdDiskFormatUnitName,
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

func renderDockerMountUnit(params diskParams) (k8scloudconfig.UnitAsset, error) {
	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: dockerMountUnitTemplate,
		Name:         dockerMountUnitName,
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

func renderDockerDiskFormatUnit(params diskParams) (k8scloudconfig.UnitAsset, error) {
	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: dockerDiskFormatUnitTemplate,
		Name:         dockerDiskFormatUnitName,
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
