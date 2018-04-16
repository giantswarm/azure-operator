package dnsrecord

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = microerror.New("invalid config")

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var wrongTypeError = microerror.New("wrong type")

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}

var timeoutError = microerror.New("timeout")

// IsTimeoutError asserts deleteTimeoutError.
func IsTimeoutError(err error) bool {
	return microerror.Cause(err) == timeoutError
}
