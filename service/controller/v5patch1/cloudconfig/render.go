package cloudconfig

import (
	"encoding/base64"

	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_4_1_1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v5patch1/encrypter"
)

func renderCalicoAzureFile(params calicoAzureFileParams) (k8scloudconfig.FileAsset, error) {
	fileMeta := k8scloudconfig.FileMetadata{
		AssetContent: calicoAzureFileTemplate,
		Path:         calicoAzureFileName,
		Owner: k8scloudconfig.Owner{
			User:  FileOwnerUser,
			Group: FileOwnerGroup,
		},
		Permissions: calicoAzureFilePermission,
	}

	content, err := k8scloudconfig.RenderFileAssetContent(fileMeta.AssetContent, params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	file := k8scloudconfig.FileAsset{
		Metadata: fileMeta,
		Content:  content,
	}

	return file, nil
}

func renderCertificatesFiles(encrypter encrypter.Interface, certFiles certs.Files) ([]k8scloudconfig.FileAsset, error) {
	var certsMeta []k8scloudconfig.FileMetadata
	for _, f := range certFiles {
		encryptedData, err := encrypter.Encrypt(f.Data)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		m := k8scloudconfig.FileMetadata{
			AssetContent: string(encryptedData),
			Path:         f.AbsolutePath + ".enc",
			Owner: k8scloudconfig.Owner{
				User:  FileOwnerUser,
				Group: FileOwnerGroup,
			},
			Permissions: certFilePermission,
		}
		certsMeta = append(certsMeta, m)
	}

	var files []k8scloudconfig.FileAsset
	for _, cm := range certsMeta {
		content := base64.StdEncoding.EncodeToString([]byte(cm.AssetContent))

		f := k8scloudconfig.FileAsset{
			Metadata: cm,
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
		Owner: k8scloudconfig.Owner{
			User:  FileOwnerUser,
			Group: FileOwnerGroup,
		},
		Permissions: cloudProviderConfFilePermission,
	}

	content, err := k8scloudconfig.RenderFileAssetContent(fileMeta.AssetContent, params)
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
		Owner: k8scloudconfig.Owner{
			User:  FileOwnerUser,
			Group: FileOwnerGroup,
		},
		Permissions: defaultStorageClassFilePermission,
	}

	content, err := k8scloudconfig.RenderFileAssetContent(fileMeta.AssetContent, params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	file := k8scloudconfig.FileAsset{
		Metadata: fileMeta,
		Content:  content,
	}

	return file, nil
}

func renderIngressLBFile(params ingressLBFileParams) (k8scloudconfig.FileAsset, error) {
	fileMeta := k8scloudconfig.FileMetadata{
		AssetContent: ingressLBFileTemplate,
		Path:         ingressLBFileName,
		Owner: k8scloudconfig.Owner{
			User:  FileOwnerUser,
			Group: FileOwnerGroup,
		},
		Permissions: ingressLBFilePermission,
	}

	content, err := k8scloudconfig.RenderFileAssetContent(fileMeta.AssetContent, params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	file := k8scloudconfig.FileAsset{
		Metadata: fileMeta,
		Content:  content,
	}

	return file, nil
}

func renderCertificateDecrypterUnit(params certificateDecrypterUnitParams) (k8scloudconfig.UnitAsset, error) {
	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: certDecrypterUnitTemplate,
		Name:         certDecrypterUnitName,
		Enabled:      true,
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

func renderEtcdMountUnit() (k8scloudconfig.UnitAsset, error) {
	params := struct{}{}

	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: etcdMountUnitTemplate,
		Name:         etcdMountUnitName,
		Enabled:      true,
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
		Enabled:      true,
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

func renderDockerMountUnit() (k8scloudconfig.UnitAsset, error) {
	params := struct{}{}

	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: dockerMountUnitTemplate,
		Name:         dockerMountUnitName,
		Enabled:      true,
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
		Enabled:      true,
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

func renderIngressLBUnit() (k8scloudconfig.UnitAsset, error) {
	params := struct{}{}

	unitMeta := k8scloudconfig.UnitMetadata{
		AssetContent: ingressLBUnitTemplate,
		Name:         ingressLBUnitName,
		Enabled:      false,
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
