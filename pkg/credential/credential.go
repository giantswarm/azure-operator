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

// GetTenantAzureClientCredentialsConfig returns ClientCredentialsConfig with the customer TenantID as auxiliary tenant id,
// so that a multi-tenant service principal can work on subscription linked to the customer Tenant ID.
// It will use a single tenant service principal if credential secret contains customer clientid and clientsecret.
func GetTenantAzureClientCredentialsConfig(k8sClient kubernetes.Interface, cr providerv1alpha1.AzureConfig, gsClientCredentialsConfig auth.ClientCredentialsConfig) (auth.ClientCredentialsConfig, error) {
	// Refactor to only decorate GS ClientCredentialsConfig?
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

	// Use GiantSwarm Multi-tenant SP
	if clientID == "" && clientSecret == "" {
		gsClientCredentialsConfig.AuxTenants = append(gsClientCredentialsConfig.AuxTenants, tenantID)
	}

	// Use customer SP if credentials are present in credential secret
	gsClientCredentialsConfig = auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)

	return gsClientCredentialsConfig, nil
}

// GetTenantAzureClientCredentialsConfig returns ClientCredentialsConfig with the customer TenantID as auxiliary tenant id,
// so that a multi-tenant service principal can work on subscription linked to the customer Tenant ID.
func GetTenantAzureClientCredentialsConfig2(gsClientCredentialsConfig auth.ClientCredentialsConfig, tenantID string) (auth.ClientCredentialsConfig, error) {
	gsClientCredentialsConfig.AuxTenants = append(gsClientCredentialsConfig.AuxTenants, tenantID)

	return gsClientCredentialsConfig, nil
}

func GetTenant(k8sClient kubernetes.Interface, cr providerv1alpha1.AzureConfig) (string, string, string, error) {
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
		// No having partnerID in the secret means that customer has not
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
