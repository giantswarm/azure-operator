package credential

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	ClientIDKey         = "azure.azureoperator.clientid"
	ClientSecretKey     = "azure.azureoperator.clientsecret"
	SubscriptionIDKey   = "azure.azureoperator.subscriptionid"
	TenantIDKey         = "azure.azureoperator.tenantid"
	PartnerIDKey        = "azure.azureoperator.partnerid"
	SecretLabel         = "giantswarm.io/managed-by=credentiald"
	CredentialNamespace = "giantswarm"
	CredentialDefault   = "credential-default"
)

func GetAzureConfigFromSecretName(k8sClient kubernetes.Interface, name string, namespace string) (*client.AzureClientSetConfig, error) {
	credential, err := k8sClient.CoreV1().Secrets(namespace).Get(name, apismetav1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return GetAzureConfigFromSecret(credential)
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
		partnerID = ""
	}

	azureClientSetConfig := client.AzureClientSetConfig{
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		PartnerID:      partnerID,
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
	}

	return &azureClientSetConfig, nil
}

func GetAzureClientSetsFromCredentialSecrets(k8sclient kubernetes.Interface) (map[*client.AzureClientSetConfig]*client.AzureClientSet, error) {
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

		clientSet, err := client.NewAzureClientSet(*azureClientSetConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		azureClientSets[azureClientSetConfig] = clientSet
	}

	return azureClientSets, nil
}

func GetAzureClientSetsFromCredentialSecretsBySubscription(k8sclient kubernetes.Interface) (map[string]*client.AzureClientSet, error) {
	azureClientSets := map[string]*client.AzureClientSet{}

	rawAzureClientSets, err := GetAzureClientSetsFromCredentialSecrets(k8sclient)
	if err != nil {
		return azureClientSets, microerror.Mask(err)
	}

	for azureClientSetConfig, azureClientSet := range rawAzureClientSets {
		azureClientSets[azureClientSetConfig.SubscriptionID] = azureClientSet
	}

	return azureClientSets, nil
}

func GetAzureClientSetsByCluster(k8sclient kubernetes.Interface, g8sclient versioned.Interface) (map[string]*client.AzureClientSet, error) {
	azureClientSets := map[string]*client.AzureClientSet{}
	var crs []providerv1alpha1.AzureConfig
	{
		mark := ""
		page := 0
		for page == 0 || len(mark) > 0 {
			opts := apismetav1.ListOptions{
				Continue: mark,
			}
			list, err := g8sclient.ProviderV1alpha1().AzureConfigs(apismetav1.NamespaceAll).List(opts)
			if err != nil {
				return azureClientSets, microerror.Mask(err)
			}

			crs = append(crs, list.Items...)

			mark = list.Continue
			page++
		}
	}

	for _, cr := range crs {
		config, err := GetAzureConfigFromSecretName(k8sclient, key.CredentialName(cr), key.CredentialNamespace(cr))
		if err != nil {
			return nil, microerror.Mask(err)
		}
		azureClients, err := client.NewAzureClientSet(*config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		azureClientSets[cr.GetName()] = azureClients
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
		list, err := k8sClient.CoreV1().Secrets(CredentialNamespace).List(opts)
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
