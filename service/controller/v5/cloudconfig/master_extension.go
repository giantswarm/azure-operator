package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_3_6_2"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
)

type masterExtension struct {
	Azure         setting.Azure
	AzureConfig   client.AzureClientSetConfig
	CalicoCIDR    string
	CertsSearcher certs.Interface
	CustomObject  providerv1alpha1.AzureConfig
}

// Files allows files to be injected into the master cloudconfig.
func (me *masterExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	calicoAzureFile, err := me.renderCalicoAzureFile()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	certificateFiles, err := me.renderCertificatesFiles()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cloudProviderConfFile, err := me.renderCloudProviderConfFile()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	defaultStorageClassFile, err := me.renderDefaultStorageClassFile()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ingressLBFile, err := me.renderIngressLBFile()
	if err != nil {
		return nil, microerror.Maskf(err, "renderIngressLBFile")
	}

	files := []k8scloudconfig.FileAsset{
		calicoAzureFile,
		cloudProviderConfFile,
		defaultStorageClassFile,
		ingressLBFile,
	}
	files = append(files, certificateFiles...)

	return files, nil
}

// Units allows systemd units to be injected into the master cloudconfig.
func (me *masterExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	// Unit to format etcd disk.
	formatEtcdUnit, err := me.renderEtcdDiskFormatUnit()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Unit to mount etcd disk.
	mountEtcdUnit, err := me.renderEtcdMountUnit()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Unit to format docker disk.
	formatDockerUnit, err := me.renderDockerDiskFormatUnit()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Unit to mount docker disk.
	mountDockerUnit, err := me.renderDockerMountUnit()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Unit to create Kubernetes load balancer.
	ingressLBUnit, err := me.renderIngressLBUnit()
	if err != nil {
		return nil, microerror.Maskf(err, "renderIngressLBUnit")
	}

	units := []k8scloudconfig.UnitAsset{
		formatEtcdUnit,
		mountEtcdUnit,
		formatDockerUnit,
		mountDockerUnit,
		ingressLBUnit,
	}

	return units, nil
}

// VerbatimSections allows sections to be embedded in the master cloudconfig.
func (me *masterExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

func (me *masterExtension) renderCalicoAzureFile() (k8scloudconfig.FileAsset, error) {
	params := newCalicoAzureFileParams(me.CustomObject, me.CalicoCIDR)

	asset, err := renderCalicoAzureFile(params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderCertificatesFiles() ([]k8scloudconfig.FileAsset, error) {
	clusterCerts, err := me.CertsSearcher.SearchCluster(key.ClusterID(me.CustomObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	assets, err := renderCertificatesFiles(certs.NewFilesClusterMaster(clusterCerts))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return assets, nil
}

func (me *masterExtension) renderCloudProviderConfFile() (k8scloudconfig.FileAsset, error) {
	params := newCloudProviderConfFileParams(me.Azure, me.AzureConfig, me.CustomObject)

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

func (me *masterExtension) renderIngressLBFile() (k8scloudconfig.FileAsset, error) {
	params := newIngressLBFileParams(me.CustomObject)

	asset, err := renderIngressLBFile(params)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderEtcdMountUnit() (k8scloudconfig.UnitAsset, error) {
	asset, err := renderEtcdMountUnit()
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderEtcdDiskFormatUnit() (k8scloudconfig.UnitAsset, error) {
	params := diskParams{
		LUNID: "0",
	}

	asset, err := renderEtcdDiskFormatUnit(params)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderDockerMountUnit() (k8scloudconfig.UnitAsset, error) {
	asset, err := renderDockerMountUnit()
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderDockerDiskFormatUnit() (k8scloudconfig.UnitAsset, error) {
	params := diskParams{
		LUNID: "1",
	}

	asset, err := renderDockerDiskFormatUnit(params)
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}

func (me *masterExtension) renderIngressLBUnit() (k8scloudconfig.UnitAsset, error) {
	asset, err := renderIngressLBUnit()
	if err != nil {
		return k8scloudconfig.UnitAsset{}, microerror.Mask(err)
	}

	return asset, nil
}
