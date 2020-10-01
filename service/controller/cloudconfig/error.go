package cloudconfig

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var invalidSecretError = &microerror.Error{
	Kind: "invalidSecretError",
}

func IsInvalidSecret(err error) bool {
	return microerror.Cause(err) == invalidSecretError
}

var timeoutError = &microerror.Error{
	Kind: "timeoutError",
}

func IsTimeout(err error) bool {
	return microerror.Cause(err) == timeoutError
}
