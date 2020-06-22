package credential

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/k8sclient/v3/pkg/k8sclient"
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

type Secret struct {
	Namespace string
	Name      string
}

func toObjectKey(credential Secret) client.ObjectKey {
	return client.ObjectKey{Namespace: credential.Namespace, Name: credential.Name}
}

// GetOrganizationAzureCredentials returns the organization's credentials.
// This means a configured `ClientCredentialsConfig` together with the subscription ID and the partner ID.
// The Service Principals in the organizations' secrets will always belong the the GiantSwarm Tenant ID in `gsTenantID`.
func GetOrganizationAzureCredentials(ctx context.Context, k8sClient k8sclient.Interface, credentialSecret Secret, gsTenantID string) (auth.ClientCredentialsConfig, string, string, error) {
	credential := &v1.Secret{}
	err := k8sClient.CtrlClient().Get(ctx, toObjectKey(credentialSecret), credential)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	return GetCredentialsFromCredentialSecret(credential, gsTenantID)
}

func GetOrganizationAzureCredentialsFromCredentialSecret(ctx context.Context, ctrlClient client.Client, credentialSecret providerv1alpha1.CredentialSecret, gsTenantID string) (auth.ClientCredentialsConfig, string, string, error) {
	credential := &v1.Secret{}
	err := ctrlClient.Get(ctx, client.ObjectKey{Namespace: credentialSecret.Namespace, Name: credentialSecret.Name}, credential)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	return GetCredentialsFromCredentialSecret(credential, gsTenantID)
}

func GetCredentialsFromCredentialSecret(credentialSecret *v1.Secret, gsTenantID string) (auth.ClientCredentialsConfig, string, string, error) {
	clientID, err := valueFromSecret(credentialSecret, clientIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	clientSecret, err := valueFromSecret(credentialSecret, clientSecretKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	tenantID, err := valueFromSecret(credentialSecret, tenantIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	subscriptionID, err := valueFromSecret(credentialSecret, subscriptionIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	partnerID, err := valueFromSecret(credentialSecret, partnerIDKey)
	if err != nil {
		// No having Partner ID in the secret means that customer has not
		// upgraded yet to use the Azure Partner Program. In that case we set a
		// constant random generated GUID that we haven't registered with Azure.
		// When all customers have migrated, we should error out instead.
		partnerID = defaultAzureGUID
	}

	if _, exists := credentialSecret.GetLabels()[label.SingleTenantSP]; exists || tenantID == gsTenantID {
		// The tenant cluster resources will belong to a subscription linked to the same Tenant ID used for authentication.
		credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
		return credentials, subscriptionID, partnerID, nil
	}

	credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, gsTenantID)
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
