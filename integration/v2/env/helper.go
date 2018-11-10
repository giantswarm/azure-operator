package env

import (
	"os"

	"github.com/giantswarm/microerror"
)

// getEnvVarOptional never returns error but for visual aligmnent it has the
// same interface as getEnvVarRequired.
func getEnvVarOptional(name string, v *string) (string, error) {
	return os.Getenv(name), nil
}

func getEnvVarRequired(name string, v *string) (string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", microerror.Maskf(executionFailedError, "env var %#q must not be empty", name)
	}

}
