package azureconfig

import (
	"github.com/giantswarm/microerror"
)

// executionFailedError is an error type for situations where Resource
// execution cannot continue and must always fall back to operatorkit.
//
// This error should never be matched against and therefore there is no matcher
// implement. For further information see:
//
//	https://github.com/giantswarm/fmt/blob/master/go/errors.md#matching-errors
var executionFailedError = &microerror.Error{
	Kind: "executionFailedError",
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var invalidDomainError = &microerror.Error{
	Kind: "invalidDomainError",
}

// IsInvalidDomain asserts invalidDomainError.
func IsInvalidDomain(err error) bool {
	return microerror.Cause(err) == invalidDomainError
}

var invalidSubnetMaskError = &microerror.Error{
	Kind: "invalidSubnetMaskError",
}

// IsInvalidSubnetMask asserts invalidSubnetMaskError.
func IsInvalidSubnetMask(err error) bool {
	return microerror.Cause(err) == invalidSubnetMaskError
}

var tooManyCredentialsError = &microerror.Error{
	Kind: "tooManyCredentialsError",
}

// IsTooManyCredentials asserts tooManyCredentialsError.
func IsTooManyCredentials(err error) bool {
	return microerror.Cause(err) == tooManyCredentialsError
}

var credentialsNotFoundError = &microerror.Error{
	Kind: "credentialsNotFoundError",
}

// IsCredentialsNotFoundError asserts credentialsNotFoundError.
func IsCredentialsNotFoundError(err error) bool {
	return microerror.Cause(err) == credentialsNotFoundError
}
