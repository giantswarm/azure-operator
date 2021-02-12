package credential

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type Provider interface {
	GetOrganizationAzureCredentials(ctx context.Context, clusterID string) (auth.ClientCredentialsConfig, string, string, error)
}
type EmptyProvider struct {
}

func (p EmptyProvider) GetOrganizationAzureCredentials(ctx context.Context, clusterID string) (auth.ClientCredentialsConfig, string, string, error) {
	return auth.ClientCredentialsConfig{}, "", "", nil
}
