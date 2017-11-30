package cloudconfig

import (
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azuretpr"
)

type keyVaultSecrets struct {
	VaultName string
	Secrets   []keyVaultSecret
}

type keyVaultSecret struct {
	SecretName string
	FileName   string
}

type cloudConfigExtension struct {
	AzureConfig  client.AzureConfig
	CustomObject azuretpr.CustomObject
}

type MasterExtension struct {
	cloudConfigExtension
}

type WorkerExtension struct {
	cloudConfigExtension
}
