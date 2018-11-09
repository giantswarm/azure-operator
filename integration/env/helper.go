package env

import (
	"os"

	"github.com/giantswarm/microerror"
)

type getEnvFunc func() (string, error)

func getEnvs(getEnvFuncs ...getEnvFunc) error {
	for _, f := range getEnvFuncs {
		v, err := f()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func getEnvOptional(name string, v *string) getEnvFunc {
	return func() (string, error) {
		return os.Getenv(name), nil
	}
}

func getEnvRequired(name string, v *string) getEnvFunc {
	return func() (string, error) {
		v := getEnv(name)
		if v == "" {
			return "", microerror.Maskf(executionFailedError, "env var %#q must not be empty", name)
		}

		return v, nil
	}
}
