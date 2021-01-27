package tenantcluster

import (
	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var apiNotAvailableError = &microerror.Error{
	Kind: "apiNotAvailableError",
}

// IsAPINotAvailableError asserts apiNotAvailableError.
func IsAPINotAvailableError(err error) bool {
	if tenant.IsAPINotAvailable(err) {
		return true
	}

	return microerror.Cause(err) == apiNotAvailableError
}
