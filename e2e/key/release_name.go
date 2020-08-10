package key

import (
	"fmt"
)

func CertsReleaseName(clusterID string) string {
	return fmt.Sprintf("e2esetup-certs-%s", clusterID)
}

func Namespace() string {
	return "giantswarm"
}

func TestAppReleaseName() string {
	return "test-app"
}

func VaultReleaseName() string {
	return "e2esetup-vault"
}
