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

func DefaultCatalogStorageURL() string {
	return "https://giantswarm.github.com/default-catalog"
}

func DefaultTestCatalogStorageURL() string {
	return "https://giantswarm.github.com/default-test-catalog"
}

func Namespace() string {
	return "giantswarm"
}

func NodeOperatorReleaseName() string {
	return "node-operator"
}

func ReleaseName() string {
	return "chart-operator"
}

func TestAppReleaseName() string {
	return "test-app"
}

func VaultReleaseName() string {
	return "e2esetup-vault"
}
