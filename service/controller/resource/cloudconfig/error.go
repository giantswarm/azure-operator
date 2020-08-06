package cloudconfig

import (
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

// executionFailedError is an error type for situations where Resource
// execution cannot continue and must always fall back to operatorkit.
//
// This error should never be matched against and therefore there is no matcher
// implement. For further information see:
//
//     https://github.com/giantswarm/fmt/blob/master/go/errors.md#matching-errors
//
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

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == notFoundError {
		return true
	}

	{
		dErr, ok := c.(autorest.DetailedError)
		if ok {
			if dErr.StatusCode == 404 {
				return true
			}
		}
	}

	return false
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}

// IsStorageAccountNotProvisioned asserts storage account not provisioned error from upstream's API message.
func IsStorageAccountNotProvisioned(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(microerror.Cause(err).Error(), "StorageAccountIsNotProvisioned")
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

var missingOrganizationLabel = &microerror.Error{
	Kind: "missingOrganizationLabel",
}

var missingReleaseVersionLabel = &microerror.Error{
	Kind: "missingReleaseVersionLabel",
}
