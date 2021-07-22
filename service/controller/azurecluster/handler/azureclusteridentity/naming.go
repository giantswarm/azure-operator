package azureclusteridentity

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	oldNamePrefix = "credential-"
	newNamePrefix = "org-credential-"
)

func newSecretName(legacySecret corev1.Secret) string {
	name := strings.TrimPrefix(legacySecret.Name, oldNamePrefix)

	return fmt.Sprintf("%s%s", newNamePrefix, name)
}
