package credential

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clientSecretKey  = "clientSecret"
	defaultAzureGUID = "37f13270-5c7a-56ff-9211-8426baaeaabd"
)

type K8SCredential struct {
	k8sclient  k8sclient.Interface
	gsTenantID string
	logger     micrologger.Logger
}

func NewK8SCredentialProvider(k8sclient k8sclient.Interface, gsTenantID string, logger micrologger.Logger) Provider {
	return K8SCredential{
		k8sclient:  k8sclient,
		gsTenantID: gsTenantID,
		logger:     logger,
	}
}

// GetOrganizationAzureCredentials returns the organization's credentials.
// This means a configured `ClientCredentialsConfig` together with the subscription ID and the partner ID.
// The Service Principals in the organizations' secrets will always belong the the GiantSwarm Tenant ID in `gsTenantID`.
func (k K8SCredential) GetOrganizationAzureCredentials(ctx context.Context, azureCluster v1alpha3.AzureCluster) (auth.ClientCredentialsConfig, string, string, error) {
	if azureCluster.Spec.SubscriptionID == "" {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Maskf(emptySubscriptionIDError, "SubscriptionID not set in AzureCluster %s/%s", azureCluster.Namespace, azureCluster.Name)
	}

	identity, err := k.getAzureClusterIdentity(ctx, azureCluster)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	subscriptionID := azureCluster.Spec.SubscriptionID
	clientID := identity.Spec.ClientID
	tenantID := identity.Spec.TenantID
	// TODO find a place to store the Partner ID
	partnerID := defaultAzureGUID

	secret := &v1.Secret{}
	err = k.k8sclient.CtrlClient().Get(ctx, client.ObjectKey{Namespace: identity.Spec.ClientSecret.Namespace, Name: identity.Spec.ClientSecret.Name}, secret)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	clientSecret, err := valueFromSecret(secret, clientSecretKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	if tenantID == k.gsTenantID {
		// The tenant cluster resources will belong to a subscription that belongs to the same Tenant ID used for authentication.
		k.logger.Debugf(ctx, "Azure subscription %#q belongs to the same tenant ID %#q that owns the service principal. Using single tenant authentication", subscriptionID, tenantID)
		credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
		return credentials, subscriptionID, partnerID, nil
	}

	// The tenant cluster resources will belong to a subscription that belongs to a different Tenant ID than the one used for authentication.
	k.logger.Debugf(ctx, "Azure subscription %#q belongs to the tenant ID %#q which is different than the Tenant ID %#q that owns the Service Principal. Using multi tenant authentication", subscriptionID, tenantID, k.gsTenantID)
	credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, k.gsTenantID)
	credentials.AuxTenants = append(credentials.AuxTenants, tenantID)

	return credentials, subscriptionID, partnerID, nil
}

func (k K8SCredential) getAzureClusterIdentity(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*v1alpha3.AzureClusterIdentity, error) {
	if azureCluster.Spec.IdentityRef == nil {
		return nil, microerror.Maskf(identityRefNotSetError, "IdentityRef not set for AzureCluster %s/%s", azureCluster.Namespace, azureCluster.Name)
	}

	azureClusterIdentity := &v1alpha3.AzureClusterIdentity{}
	err := k.k8sclient.CtrlClient().Get(ctx, client.ObjectKey{Name: azureCluster.Spec.IdentityRef.Name, Namespace: azureCluster.Spec.IdentityRef.Namespace}, azureClusterIdentity)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	k.logger.Debugf(ctx, "found azureClusterIdentity %s/%s", azureClusterIdentity.Namespace, azureClusterIdentity.Name)

	return azureClusterIdentity, nil
}

func valueFromSecret(secret *v1.Secret, key string) (string, error) {
	v, ok := secret.Data[key]
	if !ok {
		return "", microerror.Maskf(missingValueError, key)
	}

	return string(v), nil
}

// NewAzureCredentials returns a `ClientCredentialsConfig` configured taking values from Environment, but parameters
// have precedence over environment variables.
func NewAzureCredentials(clientID, clientSecret, tenantID string) (auth.ClientCredentialsConfig, error) {
	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return auth.ClientCredentialsConfig{}, microerror.Mask(err)
	}
	if clientID != "" {
		settings.Values[auth.ClientID] = clientID
	}
	if clientSecret != "" {
		settings.Values[auth.ClientSecret] = clientSecret
	}
	if tenantID != "" {
		settings.Values[auth.TenantID] = tenantID
	}

	if settings.Values[auth.ClientID] == "" || settings.Values[auth.ClientSecret] == "" || settings.Values[auth.TenantID] == "" {
		return auth.ClientCredentialsConfig{}, microerror.Maskf(invalidConfigError, "credentials must not be empty")
	}

	return settings.GetClientCredentials()
}
