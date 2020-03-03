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

// getClientSets fetches all unique Azure clients.
func getClientSets(k8sClient kubernetes.Interface, environmentName string) ([]*client.AzureClientSet, error) {
	credentialList, err := k8sClient.CoreV1().Secrets(credentialNamespace).List(metav1.ListOptions{
		LabelSelector: credentialLabelSelector,
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientSets := []*client.AzureClientSet{}

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

		clientSets = append(clientSets, clientSet)
	}

	return clientSets, nil
}
