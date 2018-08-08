package update

import "github.com/giantswarm/microerror"

var alreadyCreatedError = &microerror.Error{
	Kind: "alreadyCreatedError",
}

// IsAlreadyCreated asserts alreadyCreatedError.
func IsAlreadyCreated(err error) bool {
	return microerror.Cause(err) == alreadyCreatedError
}

var alreadyUpdatedError = &microerror.Error{
	Kind: "alreadyUpdatedError",
}

// IsAlreadyUpdated asserts alreadyUpdatedError.
func IsAlreadyUpdated(err error) bool {
	return microerror.Cause(err) == alreadyUpdatedError
}

var notCreatedError = &microerror.Error{
	Kind: "notCreatedError",
}

// IsNotCreated asserts notCreatedError.
func IsNotCreated(err error) bool {
	return microerror.Cause(err) == notCreatedError
}

var notUpdatedError = &microerror.Error{
	Kind: "notUpdatedError",
}

// IsNotUpdated asserts notUpdatedError.
func IsNotUpdated(err error) bool {
	return microerror.Cause(err) == notUpdatedError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
