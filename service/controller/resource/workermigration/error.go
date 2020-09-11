package workermigration

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

var invalidDomainError = &microerror.Error{
	Kind: "invalidDomainError",
}

// IsInvalidDomain asserts invalidDomainError.
func IsInvalidDomain(err error) bool {
	return microerror.Cause(err) == invalidDomainError
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	return c == notFoundError
}

var tooManyCredentialsError = &microerror.Error{
	Kind: "tooManyCredentialsError",
}

// IsTooManyCredentials asserts tooManyCredentialsError.
func IsTooManyCredentials(err error) bool {
	return microerror.Cause(err) == tooManyCredentialsError
}

var unknownKindError = &microerror.Error{
	Kind: "unknownKindError",
}

// IsUnknownKindError asserts unknownKindError.
func IsUnknownKindError(err error) bool {
	return microerror.Cause(err) == unknownKindError
}
