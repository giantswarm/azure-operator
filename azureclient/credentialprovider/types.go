package credentialprovider

import "github.com/Azure/go-autorest/autorest/azure/auth"

type AzureClientCredentialsConfig struct {
	ClientCredentialsConfig auth.ClientCredentialsConfig
	SubscriptionID          string
}
