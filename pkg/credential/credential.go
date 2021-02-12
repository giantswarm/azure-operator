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

	"github.com/giantswarm/azure-operator/v5/pkg/label"
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
func (k K8SCredential) GetOrganizationAzureCredentials(ctx context.Context, clusterID string) (auth.ClientCredentialsConfig, string, string, error) {
	azureClusters := &v1alpha3.AzureClusterList{}
	err := k.k8sclient.CtrlClient().List(ctx, azureClusters, client.MatchingLabels{label.Cluster: clusterID})
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	if len(azureClusters.Items) != 1 {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Maskf(azureClusterNotFoundError, "Expected 1 AzureCluster with label %s = %q, %d found", label.Cluster, clusterID, len(azureClusters.Items))
	}

	azureCluster := azureClusters.Items[0]

	if azureCluster.Spec.SubscriptionID == "" {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Maskf(subscriptionIDNotSetError, "AzureCluster %s/%s didn't have the SubscriptionID field set", azureCluster.Namespace, azureCluster.Name)
	}

	if azureCluster.Spec.IdentityRef == nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Maskf(identityRefNotSetError, "AzureCluster %s/%s didn't have the IdentityRef field set", azureCluster.Namespace, azureCluster.Name)
	}

	azureClusterIdentity := &v1alpha3.AzureClusterIdentity{}
	err = k.k8sclient.CtrlClient().Get(ctx, client.ObjectKey{Namespace: azureCluster.Spec.IdentityRef.Namespace, Name: azureCluster.Spec.IdentityRef.Name}, azureClusterIdentity)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	secret := &v1.Secret{}
	err = k.k8sclient.CtrlClient().Get(ctx, client.ObjectKey{Namespace: azureClusterIdentity.Spec.ClientSecret.Namespace, Name: azureClusterIdentity.Spec.ClientSecret.Name}, secret)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	subscriptionID := azureCluster.Spec.SubscriptionID
	clientID := azureClusterIdentity.Spec.ClientID
	tenantID := azureClusterIdentity.Spec.TenantID
	// TODO find a way to store the partnerID
	partnerID := defaultAzureGUID

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
