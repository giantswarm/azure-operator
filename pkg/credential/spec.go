package credential

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
)

type Provider interface {
	GetOrganizationAzureCredentials(ctx context.Context, azureCluster v1alpha3.AzureCluster) (auth.ClientCredentialsConfig, string, string, error)
}
type EmptyProvider struct {
}

func (p EmptyProvider) GetOrganizationAzureCredentials(ctx context.Context, azureCluster v1alpha3.AzureCluster) (auth.ClientCredentialsConfig, string, string, error) {
	return auth.ClientCredentialsConfig{}, "", "", nil
}
