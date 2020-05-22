package credential

import (
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/v4/client"
)

const (
	ClientIDKey       = "azure.azureoperator.clientid"
	ClientSecretKey   = "azure.azureoperator.clientsecret"
	SubscriptionIDKey = "azure.azureoperator.subscriptionid"
	TenantIDKey       = "azure.azureoperator.tenantid"
	PartnerIDKey      = "azure.azureoperator.partnerid"
)

func GetAzureClientSetFromSecretName(k8sClient kubernetes.Interface, name string, namespace string) (*client.AzureClientSet, error) {
	credential, err := k8sClient.CoreV1().Secrets(namespace).Get(name, apismetav1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return GetAzureClientSetFromSecret(credential)
}

func GetAzureClientSetFromSecret(credential *v1.Secret) (*client.AzureClientSet, error) {
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

	azureClientSetConfig, err := client.NewAzureClientSet(clientID, clientSecret, tenantID, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClientSetConfig, nil
}

func valueFromSecret(secret *v1.Secret, key string) (string, error) {
	v, ok := secret.Data[key]
	if !ok {
		return "", microerror.Maskf(missingValueError, key)
	}

	return string(v), nil
}
