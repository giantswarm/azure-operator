package credential

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
)

const (
	clientIDKey       = "azure.azureoperator.clientid"
	clientSecretKey   = "azure.azureoperator.clientsecret"
	defaultAzureGUID  = "37f13270-5c7a-56ff-9211-8426baaeaabd"
	partnerIDKey      = "azure.azureoperator.partnerid"
	subscriptionIDKey = "azure.azureoperator.subscriptionid"
	tenantIDKey       = "azure.azureoperator.tenantid"
)

type K8SCredential struct {
	k8sclient  k8sclient.Interface
	gsTenantID string
}

func NewK8SCredentialProvider(k8sclient k8sclient.Interface, gsTenantID string) Provider {
	return K8SCredential{
		k8sclient:  k8sclient,
		gsTenantID: gsTenantID,
	}
}

// GetOrganizationAzureCredentials returns the organization's credentials.
// This means a configured `ClientCredentialsConfig` together with the subscription ID and the partner ID.
// The Service Principals in the organizations' secrets will always belong the the GiantSwarm Tenant ID in `gsTenantID`.
func (k K8SCredential) GetOrganizationAzureCredentials(ctx context.Context, credentialNamespace, credentialName string) (auth.ClientCredentialsConfig, string, string, error) {
	secret := &v1.Secret{}
	err := k.k8sclient.CtrlClient().Get(ctx, client.ObjectKey{Namespace: credentialNamespace, Name: credentialName}, secret)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	clientID, err := valueFromSecret(secret, clientIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	clientSecret, err := valueFromSecret(secret, clientSecretKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	tenantID, err := valueFromSecret(secret, tenantIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	subscriptionID, err := valueFromSecret(secret, subscriptionIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	partnerID, err := valueFromSecret(secret, partnerIDKey)
	if err != nil {
		// No having Partner ID in the secret means that customer has not
		// upgraded yet to use the Azure Partner Program. In that case we set a
		// constant random generated GUID that we haven't registered with Azure.
		// When all customers have migrated, we should error out instead.
		partnerID = defaultAzureGUID
	}

	if _, exists := secret.GetLabels()[label.SingleTenantSP]; exists || tenantID == k.gsTenantID {
		// The tenant cluster resources will belong to a subscription linked to the same Tenant ID used for authentication.
		credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
		return credentials, subscriptionID, partnerID, nil
	}

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
