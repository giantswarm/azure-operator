package credential

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type Provider interface {
	GetOrganizationAzureCredentials(ctx context.Context, credentialNamespace, credentialName string) (auth.ClientCredentialsConfig, string, string, error)
}
