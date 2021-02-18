package credentialprovider

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"

	"github.com/giantswarm/azure-operator/v5/client/factory"
)

type CredentialProvider interface {
	GetAzureClientCredentialsConfig(ctx context.Context, clusterID string) (*factory.AzureClientCredentialsConfig, error)
	GetLegacyCredentialSecret(ctx context.Context, clusterID string) (*v1alpha1.CredentialSecret, error)
}
