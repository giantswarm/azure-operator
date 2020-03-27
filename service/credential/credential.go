package credential

import (
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
)

const (
	ClientIDKey         = "azure.azureoperator.clientid"
	ClientSecretKey     = "azure.azureoperator.clientsecret"
	SubscriptionIDKey   = "azure.azureoperator.subscriptionid"
	TenantIDKey         = "azure.azureoperator.tenantid"
	PartnerIDKey        = "azure.azureoperator.partnerid"
	defaultAzureGUID    = "37f13270-5c7a-56ff-9211-8426baaeaabd"
	SecretLabel         = "giantswarm.io/managed-by: credentiald"
	credentialNamespace = "giantswarm"
)

func GetAzureConfig(k8sClient kubernetes.Interface, name string, namespace string) (*client.AzureClientSetConfig, error) {
	credential, err := k8sClient.CoreV1().Secrets(namespace).Get(name, apismetav1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientID, err := valueFromSecret(credential, ClientIDKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientSecret, err := valueFromSecret(credential, ClientSecretKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	subscriptionID, err := valueFromSecret(credential, SubscriptionIDKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	tenantID, err := valueFromSecret(credential, TenantIDKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	partnerID, err := valueFromSecret(credential, PartnerIDKey)
	if err != nil {
		// No having partnerID in the secret means that customer has not
		// upgraded yet to use the Azure Partner Program. In that case we set a
		// constant random generated GUID that we haven't registered with Azure.
		// When all customers have migrated, we should error out instead.
		partnerID = defaultAzureGUID
	}

	c := &client.AzureClientSetConfig{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		PartnerID:      partnerID,
	}

	return c, nil
}

func GetAzureConfigFromSecret(credential *v1.Secret) (*client.AzureClientSetConfig, error) {
	clientID, err := valueFromSecret(credential, ClientIDKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientSecret, err := valueFromSecret(credential, ClientSecretKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	subscriptionID, err := valueFromSecret(credential, SubscriptionIDKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	tenantID, err := valueFromSecret(credential, TenantIDKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	partnerID, err := valueFromSecret(credential, PartnerIDKey)
	if err != nil {
		// No having partnerID in the secret means that customer has not
		// upgraded yet to use the Azure Partner Program. In that case we set a
		// constant random generated GUID that we haven't registered with Azure.
		// When all customers have migrated, we should error out instead.
		partnerID = defaultAzureGUID
	}

	c := &client.AzureClientSetConfig{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
		PartnerID:      partnerID,
	}

	return c, nil
}

func GetAzureClientSetsFromCredentialSecrets(k8sclient kubernetes.Interface, environmentName string) (map[*client.AzureClientSetConfig]*client.AzureClientSet, error) {
	azureClientSets := map[*client.AzureClientSetConfig]*client.AzureClientSet{}

	secrets, err := GetCredentialSecrets(k8sclient)
	if err != nil {
		return azureClientSets, microerror.Mask(err)
	}

	for _, secret := range secrets {
		azureClientSetConfig, err := GetAzureConfigFromSecret(&secret)
		if err != nil {
			return azureClientSets, microerror.Mask(err)
		}

		azureClientSetConfig.EnvironmentName = environmentName

		clientSet, err := client.NewAzureClientSet(*azureClientSetConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		azureClientSets[azureClientSetConfig] = clientSet
	}

	return azureClientSets, nil
}

func GetCredentialSecrets(k8sClient kubernetes.Interface) (secrets []v1.Secret, err error) {
	mark := ""
	page := 0
	for page == 0 || len(mark) > 0 {
		opts := apismetav1.ListOptions{
			Continue:      mark,
			LabelSelector: SecretLabel,
		}
		list, err := k8sClient.CoreV1().Secrets(credentialNamespace).List(opts)
		if err != nil {
			return secrets, microerror.Mask(err)
		}

		secrets = append(secrets, list.Items...)

		mark = list.Continue
		page++
	}

	return secrets, nil
}

func valueFromSecret(secret *v1.Secret, key string) (string, error) {
	v, ok := secret.Data[key]
	if !ok {
		return "", microerror.Maskf(missingValueError, key)
	}

	return string(v), nil
}
