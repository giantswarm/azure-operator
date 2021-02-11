package helpers

import (
	"github.com/giantswarm/microerror"
)

var invalidObjectError = &microerror.Error{
	Kind: "invalid object",
}

// IsInvalidObject asserts invalidObjectError.
func IsInvalidObject(err error) bool {
	return microerror.Cause(err) == invalidObjectError
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

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}
