package clusterownerreference

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

var crBeingDeletedError = &microerror.Error{
	Kind: "crBeingDeletedError",
}

// IsCRBeingDeletedError asserts crBeingDeletedError.
func IsCRBeingDeletedError(err error) bool {
	return microerror.Cause(err) == crBeingDeletedError
}
