package controllercontext

import (
	"github.com/giantswarm/microerror"
)

var invalidContextError = &microerror.Error{
	Kind: "invalidContextError",
}

// IsInvalidContext asserts invalidContextError.
func IsInvalidContext(err error) bool {
	return microerror.Cause(err) == invalidContextError
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}
