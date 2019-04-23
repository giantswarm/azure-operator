package collector

import (
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/credential"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// credentialNamespace is the namespace in which we store credentials.
	credentialNamespace = "giantswarm"

	// credentialLabelSelector is the label selector we use to retrieve credentials.
	credentialLabelSelector = "app=credentiald"
)

// getClientSets fetches all unique Azure tenant clusters clients.
func getClientSets(k8sClient kubernetes.Interface, environmentName string) ([]*client.AzureClientSet, error) {
	configs, err := getAzureClientSetConfigs(k8sClient, environmentName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var clientSets []*client.AzureClientSet
	for _, c := range configs {
		clientSet, err := client.NewAzureClientSet(*c)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		clientSets = append(clientSets, clientSet)
	}

	return clientSets, nil
}

func getAzureClientSetConfigs(k8sClient kubernetes.Interface, environmentName string) ([]*client.AzureClientSetConfig, error) {
	var credentials []corev1.Secret
	{
		var mark string
		for {
			opts := metav1.ListOptions{
				Continue:      mark,
				LabelSelector: credentialLabelSelector,
			}

			list, err := k8sClient.CoreV1().Secrets(credentialNamespace).List(opts)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			credentials = append(credentials, list.Items...)

			mark = list.Continue
			if mark == "" {
				break
			}
		}
	}

	configs := []*client.AzureClientSetConfig{}

	for _, secret := range credentials {
		c, err := credential.GetAzureConfig(k8sClient, secret.Name, secret.Namespace)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		c.EnvironmentName = environmentName

		configs = append(configs, c)
	}

	return configs, nil
}
