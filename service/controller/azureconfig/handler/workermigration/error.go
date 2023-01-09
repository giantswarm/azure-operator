package workermigration

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

var missingOrganizationLabel = &microerror.Error{
	Kind: "missingOrganizationLabel",
}

// IsMissingOrganizationLabel asserts missingOrganizationLabel.
func IsMissingOrganizationLabel(err error) bool {
	return microerror.Cause(err) == missingOrganizationLabel
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
