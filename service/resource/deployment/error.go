package deployment

import (
	"github.com/giantswarm/microerror"
)

var createTimeoutError = microerror.New("create timeout")

// IsCreateTimeoutError asserts createTimeoutError.
func IsCreateTimeoutError(err error) bool {
	return microerror.Cause(err) == createTimeoutError
}

var invalidConfigError = microerror.New("invalid config")

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var notFoundError = microerror.New("not found")

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}

var wrongTypeError = microerror.New("wrong type")

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
