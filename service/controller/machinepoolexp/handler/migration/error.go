package migration

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

var unknownKindError = &microerror.Error{
	Kind: "unknownKindError",
}

// IsUnknownKindError asserts unknownKindError.
func IsUnknownKindError(err error) bool {
	return microerror.Cause(err) == unknownKindError
}
