package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_4_1_0"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

type workerExtension struct {
	Azure         setting.Azure
	AzureConfig   client.AzureClientSetConfig
	CertsSearcher certs.Interface
	ClusterCerts  certs.Cluster
	CustomObject  providerv1alpha1.AzureConfig
	Encrypter     Encrypter
}

// Files allows files to be injected into the master cloudconfig.
func (we *workerExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	certificateFiles, err := we.renderCertificatesFiles()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cloudProviderConfFile, err := we.renderCloudProviderConfFile()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	files := []k8scloudconfig.FileAsset{
		cloudProviderConfFile,
	}
	files = append(files, certificateFiles...)

	return files, nil
}

// Units allows systemd units to be injected into the master cloudconfig.
func (we *workerExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	// Unit for decrypting certificates.
	certDecrypterUnit, err := we.renderCertificateDecrypterUnit()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Unit to format docker disk.
	formatDockerUnit, err := we.renderDockerDiskFormatUnit()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Unit to mount docker disk.
	mountDockerUnit, err := we.renderDockerMountUnit()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	units := []k8scloudconfig.UnitAsset{
		certDecrypterUnit,
		formatDockerUnit,
		mountDockerUnit,
	}

	return units, nil
}

// VerbatimSections allows sections to be embedded in the worker cloudconfig.
func (we *workerExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

func (we *workerExtension) renderCertificatesFiles() ([]k8scloudconfig.FileAsset, error) {
	certFiles := certs.NewFilesClusterWorker(we.ClusterCerts)
	assets, err := renderCertificatesFiles(we.Encrypter, certFiles)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return assets, nil
}

func (we *workerExtension) renderCloudProviderConfFile() (k8scloudconfig.FileAsset, error) {
	params := newCloudProviderConfFileParams(we.Azure, we.AzureConfig, we.CustomObject)

	asset, err := renderCloudProviderConfFile(params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (we *workerExtension) renderCertificateDecrypterUnit() (k8scloudconfig.UnitAsset, error) {
	certFiles := certs.NewFilesClusterWorker(we.ClusterCerts)
	params := newCertificateDecrypterUnitParams(certFiles)

	asset, err := renderCertificateDecrypterUnit(params)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (we *workerExtension) renderDockerMountUnit() (k8scloudconfig.UnitAsset, error) {
	asset, err := renderDockerMountUnit()
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (we *workerExtension) renderDockerDiskFormatUnit() (k8scloudconfig.UnitAsset, error) {
	params := diskParams{
		LUNID: "0",
	}

	asset, err := renderDockerDiskFormatUnit(params)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}
