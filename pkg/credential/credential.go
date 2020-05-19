package credential

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	ClientIDKey       = "azure.azureoperator.clientid"
	ClientSecretKey   = "azure.azureoperator.clientsecret"
	SubscriptionIDKey = "azure.azureoperator.subscriptionid"
	TenantIDKey       = "azure.azureoperator.tenantid"
	PartnerIDKey      = "azure.azureoperator.partnerid"
	defaultAzureGUID  = "37f13270-5c7a-56ff-9211-8426baaeaabd"
)

// GetOrganizationAzureClientCredentialsConfig returns ClientCredentialsConfig with the organization Tenant ID as auxiliary tenant id,
// so that a Multi-Tenant Service Principal can work on the subscription linked to the organization Tenant ID.
// It will fallback to a Single Tenant Service Principal when organization's credential secret contains Client ID and Client Secret.
func GetOrganizationAzureClientCredentialsConfig(k8sClient kubernetes.Interface, cr providerv1alpha1.AzureConfig, gsClientCredentialsConfig auth.ClientCredentialsConfig) (auth.ClientCredentialsConfig, error) {
	credential, err := k8sClient.CoreV1().Secrets(key.CredentialNamespace(cr)).Get(key.CredentialName(cr), apismetav1.GetOptions{})
	if err != nil {
		return auth.ClientCredentialsConfig{}, microerror.Mask(err)
	}

	clientID, _ := valueFromSecret(credential, ClientIDKey)
	clientSecret, _ := valueFromSecret(credential, ClientSecretKey)
	tenantID, err := valueFromSecret(credential, TenantIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, microerror.Mask(err)
	}

	// Use SP defined in Organization if credentials are present in credential secret.
	// This will happen until customer have migrated to the Multi-Tenant Service Principal.
	if clientID != "" && clientSecret != "" {
		return auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID), nil
	}

	// Use GiantSwarm Multi-tenant SP otherwise.
	gsClientCredentialsConfig.AuxTenants = append(gsClientCredentialsConfig.AuxTenants, tenantID)

	return gsClientCredentialsConfig, nil
}

// GetOrganizationTenant returns the information that makes up the organization's Azure Tenant: a Tenant ID, a Subscription ID and a Partner ID.
func GetOrganizationTenant(k8sClient kubernetes.Interface, cr providerv1alpha1.AzureConfig) (string, string, string, error) {
	credential, err := k8sClient.CoreV1().Secrets(key.CredentialNamespace(cr)).Get(key.CredentialName(cr), apismetav1.GetOptions{})
	if err != nil {
		return "", "", "", microerror.Mask(err)
	}

	tenantID, err := valueFromSecret(credential, TenantIDKey)
	if err != nil {
		return "", "", "", microerror.Mask(err)
	}

	subscriptionID, err := valueFromSecret(credential, SubscriptionIDKey)
	if err != nil {
		return "", "", "", microerror.Mask(err)
	}

	partnerID, err := valueFromSecret(credential, PartnerIDKey)
	if err != nil {
		// No having Partner ID in the secret means that customer has not
		// upgraded yet to use the Azure Partner Program. In that case we set a
		// constant random generated GUID that we haven't registered with Azure.
		// When all customers have migrated, we should error out instead.
		partnerID = defaultAzureGUID
	}

	return tenantID, subscriptionID, partnerID, nil
}

func GetSubscriptionAndPartnerID(k8sClient kubernetes.Interface, cr providerv1alpha1.AzureConfig) (string, string, error) {
	credential, err := k8sClient.CoreV1().Secrets(key.CredentialNamespace(cr)).Get(key.CredentialName(cr), apismetav1.GetOptions{})
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	subscriptionID, err := valueFromSecret(credential, SubscriptionIDKey)
	if err != nil {
		return "", "", microerror.Mask(err)
	}

	partnerID, err := valueFromSecret(credential, PartnerIDKey)
	if err != nil {
		// No having partnerID in the secret means that customer has not
		// upgraded yet to use the Azure Partner Program. In that case we set a
		// constant random generated GUID that we haven't registered with Azure.
		// When all customers have migrated, we should error out instead.
		partnerID = defaultAzureGUID
	}

	return subscriptionID, partnerID, nil
}

func valueFromSecret(secret *v1.Secret, key string) (string, error) {
	v, ok := secret.Data[key]
	if !ok {
		return "", microerror.Maskf(missingValueError, key)
	}

	return string(v), nil
}
