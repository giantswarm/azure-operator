package credentialprovider

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
)

// The CredentialProvider interface defines an interface for a module that is able to retrieve Azure API Credentials.
type CredentialProvider interface {
	// Returns the azure credentials alongside the subscription ID.
	GetAzureClientCredentialsConfig(ctx context.Context, clusterID string) (*AzureClientCredentialsConfig, error)

	// Retrieves the Legacy GS secret for an organization.
	GetLegacyCredentialSecret(ctx context.Context, organizationID string) (*v1alpha1.CredentialSecret, error)
}
