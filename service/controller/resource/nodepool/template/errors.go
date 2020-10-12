package template

import (
	"github.com/giantswarm/microerror"
)

var invalidValueError = &microerror.Error{
	Kind: "invalidValueError",
}

// IsInvalidValueError asserts invalidValueError.
func IsInvalidValueError(err error) bool {
	return microerror.Cause(err) == invalidValueError
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
