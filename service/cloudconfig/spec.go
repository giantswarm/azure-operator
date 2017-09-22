package cloudconfig

import "github.com/giantswarm/azuretpr"

type keyVaultSecrets struct {
	VaultName string
	Secrets   []keyVaultSecret
}

type keyVaultSecret struct {
	SecretName string
	FileName   string
}

type CloudConfigExtension struct {
	CustomObject azuretpr.CustomObject
}

type MasterExtension struct {
	CloudConfigExtension
}

type WorkerExtension struct {
	CloudConfigExtension
}
