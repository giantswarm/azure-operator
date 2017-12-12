package cloudconfig

import (
	"github.com/giantswarm/azure-operator/client"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
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
	CustomObject providerv1alpha1.AzureConfig
}

type masterExtension struct {
	cloudConfigExtension
}

type workerExtension struct {
	cloudConfigExtension
}
