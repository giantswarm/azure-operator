package cloudconfig

import (
	k8scloudconfig "github.com/giantswarm/k8scloudconfig"
)


// Files allows files to be injected into the master cloudconfig. It is
// currently empty because the certificates will be stored in Key Vault.
func (me *MasterExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	return nil, nil
}

// Units allows systemd units to be injected into the master cloudconfig.
// TODO Add unit for downloading certificates from Key Vault.
func (me *MasterExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	return nil, nil
}

// VerbatimSections allows sections to be embedded in the master cloudconfig.
func (me *MasterExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}


// Files allows files to be injected into the worker cloudconfig. It is
// currently empty because the certificates will be stored in Key Vault.
func (we *WorkerExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	return nil, nil
}

// Units allows systemd units to be injected into the worker cloudconfig.
// TODO Add unit for downloading certificates from Key Vault.
func (we *WorkerExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	return nil, nil
}

// VerbatimSections allows sections to be embedded in the worker cloudconfig.
func (we *WorkerExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}
