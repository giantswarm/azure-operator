package credential

import (
	"github.com/giantswarm/microerror"
	"k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

const (
	ClientIDKey       = "clientID"
	ClientSecretKey   = "clientSecret"
	SubscriptionIDKey = "subscriptionID"
	TenantIDKey       = "tenantID"
)

func GetAzureConfig(k8sClient kubernetes.Interface, obj interface{}) (*client.AzureClientSetConfig, error) {
	credential, err := readCredential(k8sClient, obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	config, err := getAzureConfig(credential)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return config, nil
}

func getAzureConfig(credential *v1.Secret) (*client.AzureClientSetConfig, error) {
	errorFormat := "%s not found in credential"

	clientID, ok := credential.Data[ClientIDKey]
	if !ok {
		return nil, microerror.Maskf(invalidConfig, errorFormat, ClientIDKey)
	}

	clientSecret, ok := credential.Data[ClientSecretKey]
	if !ok {
		return nil, microerror.Maskf(invalidConfig, errorFormat, ClientSecretKey)
	}

	subscriptionID, ok := credential.Data[SubscriptionIDKey]
	if !ok {
		return nil, microerror.Maskf(invalidConfig, errorFormat, SubscriptionIDKey)
	}

	tenantID, ok := credential.Data[TenantIDKey]
	if !ok {
		return nil, microerror.Maskf(invalidConfig, errorFormat, TenantIDKey)
	}

	c := &client.AzureClientSetConfig{
		ClientID:       string(clientID),
		ClientSecret:   string(clientSecret),
		SubscriptionID: string(subscriptionID),
		TenantID:       string(tenantID),
	}

	return c, nil
}

func readCredential(k8sClient kubernetes.Interface, obj interface{}) (*v1.Secret, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	credentialName := key.CredentialName(customObject)
	credentialNamespace := key.CredentialNamespace(customObject)

	credential, err := k8sClient.CoreV1().Secrets(credentialNamespace).Get(credentialName, apismetav1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return credential, nil
}
