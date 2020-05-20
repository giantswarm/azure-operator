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
	auxiliaryTenantIDKey = "azure.azureoperator.auxiliarytenantid"
	clientIDKey          = "azure.azureoperator.clientid"
	clientSecretKey      = "azure.azureoperator.clientsecret"
	defaultAzureGUID     = "37f13270-5c7a-56ff-9211-8426baaeaabd"
	partnerIDKey         = "azure.azureoperator.partnerid"
	singleTenantSPLabel  = "azure-operator.giantswarm.io/single-tenant-sp"
	subscriptionIDKey    = "azure.azureoperator.subscriptionid"
	tenantIDKey          = "azure.azureoperator.tenantid"
)

// GetOrganizationAzureCredentials returns the organization's credentials.
// This means a configured `ClientCredentialsConfig` together with the subscription ID and the partner ID.
// We use a label to identify organizations' secrets that contain Service Principals to still be used as Single Tenant.
// The Service Principals in the organizations' secrets will always belong the the GiantSwarm Tenant ID in `gsTenantID`.
func GetOrganizationAzureCredentials(k8sClient kubernetes.Interface, cr providerv1alpha1.AzureConfig, gsTenantID string) (auth.ClientCredentialsConfig, string, string, error) {
	credential, err := k8sClient.CoreV1().Secrets(key.CredentialNamespace(cr)).Get(key.CredentialName(cr), apismetav1.GetOptions{})
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	clientID, err := valueFromSecret(credential, clientIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	clientSecret, err := valueFromSecret(credential, clientSecretKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	tenantID, err := valueFromSecret(credential, tenantIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	subscriptionID, err := valueFromSecret(credential, subscriptionIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	partnerID, err := valueFromSecret(credential, partnerIDKey)
	if err != nil {
		// No having Partner ID in the secret means that customer has not
		// upgraded yet to use the Azure Partner Program. In that case we set a
		// constant random generated GUID that we haven't registered with Azure.
		// When all customers have migrated, we should error out instead.
		partnerID = defaultAzureGUID
	}

	// Having the label means that the Service Principal is single tenant.
	// The tenant cluster resources will belong to a subscription linked to the same Tenant ID used for authentication.
	if isSingleTenantServicePrincipal(credential) {
		credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
		return credentials, subscriptionID, partnerID, nil
	}

	auxiliaryTenantID, err := valueFromSecret(credential, auxiliaryTenantIDKey)
	if err != nil {
		return auth.ClientCredentialsConfig{}, "", "", microerror.Mask(err)
	}

	credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, gsTenantID)
	credentials.AuxTenants = append(credentials.AuxTenants, auxiliaryTenantID)

	return credentials, subscriptionID, partnerID, nil
}

func isSingleTenantServicePrincipal(secret *v1.Secret) bool {
	_, exists := secret.GetLabels()[singleTenantSPLabel]
	return exists
}

func valueFromSecret(secret *v1.Secret, key string) (string, error) {
	v, ok := secret.Data[key]
	if !ok {
		return "", microerror.Maskf(missingValueError, key)
	}

	return string(v), nil
}
