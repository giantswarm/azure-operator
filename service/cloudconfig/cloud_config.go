package cloudconfig

import (
	k8scloudconfig "github.com/giantswarm/k8scloudconfig"
)

type MasterExtension struct{}

func (me *MasterExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	return nil, nil
}

func (me *MasterExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	return nil, nil
}

func (me *MasterExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

type WorkerExtension struct{}

func (we *WorkerExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	return nil, nil
}

func (we *WorkerExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	return nil, nil
}

func (we *WorkerExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}
