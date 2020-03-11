package collector

import (
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/credential"
)

const (
	// credentialNamespace is the namespace in which we store credentials.
	credentialNamespace = "giantswarm"

	// credentialLabelSelector is the label selector we use to retrieve credentials.
	credentialLabelSelector = "app=credentiald"
)

// getClientSets fetches all Azure clients, grouped by subscription id.
// If two secrets use the same subscription but different client id, only one is returned
func getClientSets(k8sClient kubernetes.Interface, environmentName string) (map[string]*client.AzureClientSet, error) {
	credentialList, err := k8sClient.CoreV1().Secrets(credentialNamespace).List(metav1.ListOptions{
		LabelSelector: credentialLabelSelector,
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientSets := map[string]*client.AzureClientSet{}

	for _, secret := range credentialList.Items {
		config, err := credential.GetAzureConfig(k8sClient, secret.Name, secret.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		config.EnvironmentName = environmentName

		clientSet, err := client.NewAzureClientSet(*config)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		clientSets[config.SubscriptionID] = clientSet
	}

	return clientSets, nil
}
