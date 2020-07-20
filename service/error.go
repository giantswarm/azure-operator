package service
//dummy PR
import (
	"github.com/giantswarm/microerror"
)
//dummy PR 2
var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
