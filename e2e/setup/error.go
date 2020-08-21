package setup

import (
	"github.com/giantswarm/microerror"
)

var invalidAppVersionError = &microerror.Error{
	Kind: "invalidAppVersionError",
}

var idSpaceExhaustedError = &microerror.Error{
	Kind: "idSpaceExhaustedError",
}

// IsIDSpaceExhausted asserts idSpaceExhaustedError.
func IsIDSpaceExhausted(err error) bool {
	return microerror.Cause(err) == idSpaceExhaustedError
}
