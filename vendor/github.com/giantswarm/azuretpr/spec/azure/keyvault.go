package azure

type KeyVault struct {
	// Name is the name of the Azure Key Vault. It must be globally unique,
	// 3-24 characters in length and contain only (0-9, a-z, A-Z, and -).
	Name string `json:"name" yaml:"name"`
}
