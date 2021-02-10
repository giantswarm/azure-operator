package azureclusteridentity

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
)

const (
	oldNamePrefix = "credential-"
	newNamePrefix = "org-credential-"
)

func newSecretName(legacySecret corev1.Secret) string {
	name := strings.TrimPrefix(legacySecret.Name, oldNamePrefix)

	return fmt.Sprintf("%s%s", newNamePrefix, name)
}

func newSecretNamespace(legacySecret corev1.Secret) string {
	return legacySecret.Namespace
}

func legacySecretName(identity v1alpha3.AzureClusterIdentity) string {
	name := strings.TrimPrefix(identity.Name, newNamePrefix)

	return fmt.Sprintf("%s%s", oldNamePrefix, name)
}

func legacySecretNamespace(identity v1alpha3.AzureClusterIdentity) string {
	return identity.Namespace
}
