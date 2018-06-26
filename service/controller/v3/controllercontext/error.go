package controllercontext

import (
	"github.com/giantswarm/microerror"
)

var notFoundError = microerror.New("not found")

// IsNotFoundError asserts notFoundError.
func IsNotFoundError(err error) bool {
	return microerror.Cause(err) == notFoundError
}
