package cloudconfig

const (
	fileOwner                  = "root:root"
	filePermission             = 0700
	getKeyVaultSecretsFileName = "/opt/bin/get-keyvault-secrets"
	getKeyVaultSecretsTemplate = `#!/bin/bash -e

KEY_VAULT_HOST={{ .VaultName }}.vault.azure.net
AUTH_TOKEN=$(curl http://localhost:50342/oauth2/token --data "resource=https://vault.azure.net" -H Metadata:true | jq -r .access_token)
API_VERSION=2016-10-01

mkdir -p /etc/kubernetes/ssl/calico
mkdir -p /etc/kubernetes/ssl/etcd

{{ range $secret := .Secrets }}
curl https://$KEY_VAULT_HOST/secrets/{{ $secret.SecretName }}?api-version=$API_VERSION -H "Authorization: Bearer $AUTH_TOKEN" | jq -r .value > {{ $secret.FileName }} {{ end }}

chmod 0600 -R /etc/kubernetes/ssl/
chown -R etcd:etcd /etc/kubernetes/ssl/etcd/
`

	getKeyVaultSecretsUnitName = "get-keyvault-secrets.service"
	getKeyVaultSecretsUnit     = `
[Unit]
Description=Download certificates from Key Vault

[Service]
Type=oneshot
After=waagent.service
Requires=waagent.service
ExecStart=/opt/bin/get-keyvault-secrets

[Install]
WantedBy=multi-user.target
`
)
