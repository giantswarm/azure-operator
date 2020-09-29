package credential

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Provider interface {
	GetOrganizationAzureCredentials(ctx context.Context, objectMeta *v1.ObjectMeta) (auth.ClientCredentialsConfig, string, string, error)
}
type EmptyProvider struct {
}

func (p EmptyProvider) GetOrganizationAzureCredentials(ctx context.Context, objectMeta *v1.ObjectMeta) (auth.ClientCredentialsConfig, string, string, error) {
	return auth.ClientCredentialsConfig{}, "", "", nil
}
