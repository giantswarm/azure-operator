package deployment

// Deployment defines an Azure Deployment that deploys an ARM template.
type Deployment struct {
	Name          string
	Parameters    map[string]interface{}
	ResourceGroup string
	TemplateURI   string

	// TemplateContentVersion is a value to fill in
	// github.com/Azure/azure-sdk-for-go/arm/resources/resources.TemplateLink.ContentVersion.
	// For more information see contentVersion documentation
	// https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-authoring-templates.
	TemplateContentVersion string
}

// keyVaultSecrets is used to pass secrets to Key Vault as a secure object.
type keyVaultSecrets struct {
	Secrets []keyVaultSecret `json:"secrets"`
}

// keyVaultSecret is a secret stored in Key Vault.
type keyVaultSecret struct {
	SecretName  string `json:"secretName"`
	SecretValue string `json:"secretValue"`
}
