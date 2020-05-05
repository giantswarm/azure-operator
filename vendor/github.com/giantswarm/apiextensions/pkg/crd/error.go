package crd

import "github.com/giantswarm/microerror"

// conversionFailedError indicates that interface{} to versioned CRD conversion failed
var conversionFailedError = &microerror.Error{
	Kind: "conversionFailedError",
}

// IsConversionFailed asserts invalidConfigError.
func IsConversionFailed(err error) bool {
	return microerror.Cause(err) == conversionFailedError
}

// notFoundError indicates that the CRD was not found in the filesystem
var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}
