package env

import (
	"fmt"
	"os"
)

const (
	DefaultCommonDomain = "godsmack.westeurope.azure.gigantic.io"

	EnvVarCommonDomain       = "COMMON_DOMAIN"
	EnvVarRegistryPullSecret = "REGISTRY_PULL_SECRET" // nolint:gosec
	EnvVarVaultToken         = "VAULT_TOKEN"
)

var (
	commonDomain       string
	registryPullSecret string
	vaultToken         string
)

func init() {
	commonDomain = os.Getenv(EnvVarCommonDomain)
	if commonDomain == "" {
		commonDomain = DefaultCommonDomain
		fmt.Printf("No value found in '%s': using default value %s\n", EnvVarCommonDomain, DefaultCommonDomain)
	}

	registryPullSecret = os.Getenv(EnvVarRegistryPullSecret)
	if registryPullSecret == "" {
		panic(fmt.Sprintf("No value found in '%s'\n", EnvVarRegistryPullSecret))
	}

	vaultToken = os.Getenv(EnvVarVaultToken)
	if vaultToken == "" {
		vaultToken = "token"
		fmt.Printf("No value found in '%s': using default value\n", EnvVarVaultToken)
	}
}

func CommonDomain() string {
	return commonDomain
}

func RegistryPullSecret() string {
	return registryPullSecret
}

func VaultToken() string {
	return vaultToken
}
