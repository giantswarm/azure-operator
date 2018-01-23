package cloudconfig

import (
	"github.com/giantswarm/microerror"
)

var invalidCustomObjectError = microerror.New("invalid custom object")

// IsInvalidCustomObject asserts invalidCustomObjectError.
func IsInvalidCustomObject(err error) bool {
	return microerror.Cause(err) == invalidCustomObjectError
}

var invalidConfigError = microerror.New("invalid config")

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
