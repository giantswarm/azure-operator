package vnetpeering

import (
	"github.com/giantswarm/microerror"
)

var (
	invalidConfigError = microerror.New("invalid config")
	wrongTypeError     = microerror.New("wrong type")
)

func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
