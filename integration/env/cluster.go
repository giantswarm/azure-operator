package env

import (
	"fmt"
	"os"
)

const (
	DefaultCommonDomain = "godsmack.westeurope.azure.gigantic.io"

	EnvVarCommonDomain = "COMMON_DOMAIN"
	EnvVarVaultToken   = "VAULT_TOKEN"
)

var (
	commonDomain string
	vaultToken   string
)

func init() {
	commonDomain = os.Getenv(EnvVarCommonDomain)
	if commonDomain == "" {
		commonDomain = DefaultCommonDomain
		fmt.Printf("No value found in '%s': using default value %s\n", EnvVarCommonDomain, DefaultCommonDomain)
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

func VaultToken() string {
	return vaultToken
}
