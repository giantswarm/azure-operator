package microstorage

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

// NotFoundError is exported as it is used by the interface implementations
// in order to fulfil the API.
var NotFoundError = &microerror.Error{
	Kind: "NotFoundError",
}

// IsNotFound asserts NotFoundError. The library user's code should use this
// public key matcher to verify if some storage error is of type NotFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == NotFoundError
}

// InvalidKeyError is exported as it is used by the interface implementations
// in order to fulfil the API.
var InvalidKeyError = &microerror.Error{
	Kind: "InvalidKeyError",
}

// IsInvalidKey asserts InvalidKeyError. The library user's code should use
// this public key matcher to verify if some storage error is of type
// InvalidKeyError.
func IsInvalidKey(err error) bool {
	return microerror.Cause(err) == InvalidKeyError
}
