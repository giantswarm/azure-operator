package key

import (
	"fmt"
)

func CertOperatorReleaseName() string {
	return "cert-operator"
}

func CertsReleaseName(clusterID string) string {
	return fmt.Sprintf("e2esetup-certs-%s", clusterID)
}

func NodeOperatorReleaseName() string {
	return "node-operator"
}

func VaultReleaseName() string {
	return "e2esetup-vault"
}
