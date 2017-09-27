package cloudconfig

import (
	"github.com/giantswarm/certificatetpr"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/key"
)

// Files allows files to be injected into the master cloudconfig.
func (me *MasterExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	var newFiles []k8scloudconfig.FileAsset

	getSecretsScript, err := me.getMasterSecretsScript()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	newFiles = []k8scloudconfig.FileAsset{
		getSecretsScript,
	}

	return newFiles, nil
}

// Units allows systemd units to be injected into the master cloudconfig.
func (me *MasterExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	unitsMeta := []k8scloudconfig.UnitMetadata{
		{
			AssetContent: getKeyVaultSecretsUnit,
			Name:         "get-keyvault-secrets.service",
			Enable:       true,
			Command:      "start",
		},
	}

	units, err := me.renderUnits(unitsMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return units, nil
}

// VerbatimSections allows sections to be embedded in the master cloudconfig.
func (me *MasterExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

// getMasterSecretsScript returns the script for downloading the master TLS
// certificates from Key Vault on startup.
func (me *MasterExtension) getMasterSecretsScript() (k8scloudconfig.FileAsset, error) {
	secrets := keyVaultSecrets{
		VaultName: key.KeyVaultName(me.CustomObject),
		Secrets: []keyVaultSecret{
			// Kubernetes API server.
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.APIComponent, certificatetpr.CA),
				FileName:   "/etc/kubernetes/ssl/apiserver-ca.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.APIComponent, certificatetpr.Crt),
				FileName:   "/etc/kubernetes/ssl/apiserver-crt.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.APIComponent, certificatetpr.Key),
				FileName:   "/etc/kubernetes/ssl/apiserver-key.pem",
			},
			// Calico client.
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.CalicoComponent, certificatetpr.CA),
				FileName:   "/etc/kubernetes/ssl/calico/client-ca.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.CalicoComponent, certificatetpr.Crt),
				FileName:   "/etc/kubernetes/ssl/calico/client-crt.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.CalicoComponent, certificatetpr.Key),
				FileName:   "/etc/kubernetes/ssl/calico/client-key.pem",
			},
			// Etcd client.
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.EtcdComponent, certificatetpr.CA),
				FileName:   "/etc/kubernetes/ssl/etcd/client-ca.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.EtcdComponent, certificatetpr.Crt),
				FileName:   "/etc/kubernetes/ssl/etcd/client-crt.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.EtcdComponent, certificatetpr.Key),
				FileName:   "/etc/kubernetes/ssl/etcd/client-key.pem",
			},
			// Etcd server.
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.EtcdComponent, certificatetpr.CA),
				FileName:   "/etc/kubernetes/ssl/etcd/server-ca.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.EtcdComponent, certificatetpr.Crt),
				FileName:   "/etc/kubernetes/ssl/etcd/server-crt.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.EtcdComponent, certificatetpr.Key),
				FileName:   "/etc/kubernetes/ssl/etcd/server-key.pem",
			},
			// Service account.
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.ServiceAccountComponent, certificatetpr.CA),
				FileName:   "/etc/kubernetes/ssl/service-account-ca.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.ServiceAccountComponent, certificatetpr.Crt),
				FileName:   "/etc/kubernetes/ssl/service-account-crt.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.ServiceAccountComponent, certificatetpr.Key),
				FileName:   "/etc/kubernetes/ssl/service-account-key.pem",
			},
		},
	}

	getSecretsScript, err := me.renderGetSecretsScript(secrets)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return getSecretsScript, nil
}

// Files allows files to be injected into the worker cloudconfig.
func (we *WorkerExtension) Files() ([]k8scloudconfig.FileAsset, error) {
	var newFiles []k8scloudconfig.FileAsset

	getSecretsScript, err := we.getWorkerSecretsScript()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	newFiles = []k8scloudconfig.FileAsset{
		getSecretsScript,
	}

	return newFiles, nil
}

// Units allows systemd units to be injected into the worker cloudconfig.
func (we *WorkerExtension) Units() ([]k8scloudconfig.UnitAsset, error) {
	unitsMeta := []k8scloudconfig.UnitMetadata{
		{
			AssetContent: getKeyVaultSecretsUnit,
			Name:         getKeyVaultSecretsUnitName,
			Enable:       true,
			Command:      "start",
		},
	}

	units, err := we.renderUnits(unitsMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return units, nil
}

// VerbatimSections allows sections to be embedded in the worker cloudconfig.
func (we *WorkerExtension) VerbatimSections() []k8scloudconfig.VerbatimSection {
	return nil
}

// getWorkerSecretsScript returns the script for downloading the worker TLS
// certificates from Key Vault on startup.
func (we *WorkerExtension) getWorkerSecretsScript() (k8scloudconfig.FileAsset, error) {
	secrets := keyVaultSecrets{
		VaultName: key.KeyVaultName(we.CustomObject),
		Secrets: []keyVaultSecret{
			// Calico client.
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.CalicoComponent, certificatetpr.CA),
				FileName:   "/etc/kubernetes/ssl/calico/client-ca.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.CalicoComponent, certificatetpr.Crt),
				FileName:   "/etc/kubernetes/ssl/calico/client-crt.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.CalicoComponent, certificatetpr.Key),
				FileName:   "/etc/kubernetes/ssl/calico/client-key.pem",
			},
			// Etcd client.
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.EtcdComponent, certificatetpr.CA),
				FileName:   "/etc/kubernetes/ssl/etcd/client-ca.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.EtcdComponent, certificatetpr.Crt),
				FileName:   "/etc/kubernetes/ssl/etcd/client-crt.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.EtcdComponent, certificatetpr.Key),
				FileName:   "/etc/kubernetes/ssl/etcd/client-key.pem",
			},
			// Kubernetes worker.
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.WorkerComponent, certificatetpr.CA),
				FileName:   "/etc/kubernetes/ssl/worker-ca.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.WorkerComponent, certificatetpr.Crt),
				FileName:   "/etc/kubernetes/ssl/worker-crt.pem",
			},
			keyVaultSecret{
				SecretName: key.SecretName(certificatetpr.WorkerComponent, certificatetpr.Key),
				FileName:   "/etc/kubernetes/ssl/worker-key.pem",
			},
		},
	}
	getSecretsScript, err := we.renderGetSecretsScript(secrets)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	return getSecretsScript, nil
}

func (c *CloudConfigExtension) renderGetSecretsScript(secrets keyVaultSecrets) (k8scloudconfig.FileAsset, error) {
	secretsMeta := k8scloudconfig.FileMetadata{
		AssetContent: getKeyVaultSecretsTemplate,
		Path:         getKeyVaultSecretsFileName,
		Owner:        fileOwner,
		Permissions:  filePermission,
	}

	content, err := k8scloudconfig.RenderAssetContent(secretsMeta.AssetContent, secrets)
	if err != nil {
		return k8scloudconfig.FileAsset{}, microerror.Mask(err)
	}

	downloadSecrets := k8scloudconfig.FileAsset{
		Metadata: secretsMeta,
		Content:  content,
	}

	return downloadSecrets, nil
}

func (c *CloudConfigExtension) renderUnits(unitsMeta []k8scloudconfig.UnitMetadata) ([]k8scloudconfig.UnitAsset, error) {
	units := make([]k8scloudconfig.UnitAsset, 0, len(unitsMeta))

	for _, unitMeta := range unitsMeta {
		content, err := k8scloudconfig.RenderAssetContent(unitMeta.AssetContent, c.CustomObject)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		unitAsset := k8scloudconfig.UnitAsset{
			Metadata: unitMeta,
			Content:  content,
		}

		units = append(units, unitAsset)
	}

	return units, nil
}
