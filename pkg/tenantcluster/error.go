package tenantcluster

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

var APINotAvailableError = &microerror.Error{
	Kind: "APINotAvailableError",
}

// IsAPINotAvailableError asserts APINotAvailableError.
func IsAPINotAvailableError(err error) bool {
	return microerror.Cause(err) == APINotAvailableError
}
