package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_3_6_0"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v4/key"
)

type workerExtension struct {
	Azure         setting.Azure
	AzureConfig   client.AzureClientSetConfig
	CertsSearcher certs.Interface
	CustomObject  providerv1alpha1.AzureConfig
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
	clusterCerts, err := we.CertsSearcher.SearchCluster(key.ClusterID(we.CustomObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	assets, err := renderCertificatesFiles(certs.NewFilesClusterWorker(clusterCerts))
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
