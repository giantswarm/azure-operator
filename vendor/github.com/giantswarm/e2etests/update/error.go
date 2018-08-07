package update

import "github.com/giantswarm/microerror"

var alreadyUpdatedError = &microerror.Error{
	Kind: "alreadyUpdatedError",
}

// IsAlreadyUpdated asserts alreadyUpdatedError.
func IsAlreadyUpdated(err error) bool {
	return microerror.Cause(err) == alreadyUpdatedError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
