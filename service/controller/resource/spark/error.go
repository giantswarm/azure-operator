package spark

import (
	"github.com/giantswarm/microerror"
)

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

var requirementsNotMetError = &microerror.Error{
	Kind: "requirementsNotMetError",
}

// IsRequirementsNotMet asserts requirementsNotMetError.
func IsRequirementsNotMet(err error) bool {
	return microerror.Cause(err) == requirementsNotMetError
}

var timeoutError = &microerror.Error{
	Kind: "timeoutError",
}

// IsTimeout asserts timeoutError.
func IsTimeout(err error) bool {
	return microerror.Cause(err) == timeoutError
}

var tooManyCredentialsError = &microerror.Error{
	Kind: "tooManyCredentialsError",
}

// IsTooManyCredentialsError asserts tooManyCredentialsError.
func IsTooManyCredentialsError(err error) bool {
	return microerror.Cause(err) == tooManyCredentialsError
}

var missingOrganizationLabel = &microerror.Error{
	Kind: "missingOrganizationLabel",
}

// IsMissingOrganizationLabel asserts missingOrganizationLabel.
func IsMissingOrganizationLabel(err error) bool {
	return microerror.Cause(err) == missingOrganizationLabel
}

var unknownKindError = &microerror.Error{
	Kind: "unknownKindError",
}

// IsUnknownKindError asserts unknownKindError.
func IsUnknownKindError(err error) bool {
	return microerror.Cause(err) == unknownKindError
}

var credentialsNotFoundError = &microerror.Error{
	Kind: "credentialsNotFoundError",
}

// IsCredentialsNotFoundError asserts credentialsNotFoundError.
func IsCredentialsNotFoundError(err error) bool {
	return microerror.Cause(err) == credentialsNotFoundError
}
