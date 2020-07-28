package nodepool

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

var clientNotFoundError = &microerror.Error{
	Kind: "clientNotFoundError",
}

// IsClientNotFound asserts clientNotFoundError.
func IsClientNotFound(err error) bool {
	return microerror.Cause(err) == clientNotFoundError
}

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

var deploymentNotFoundError = &microerror.Error{
	Kind: "deploymentNotFoundError",
}

// IsDeploymentNotFound asserts deploymentNotFoundError.
func IsDeploymentNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == deploymentNotFoundError {
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

// IsNotFound asserts notFoundError or 404 response.
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

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

var tooManyCredentialsError = &microerror.Error{
	Kind: "tooManyCredentialsError",
}

var missingOrganizationLabel = &microerror.Error{
	Kind: "missingOrganizationLabel",
}

var missingOperatorVersionLabel = &microerror.Error{
	Kind: "missingOperatorVersionLabel",
}

var missingReleaseVersionLabel = &microerror.Error{
	Kind: "missingReleaseVersionLabel",
}

var notAvailableFailureDomain = &microerror.Error{
	Kind: "notAvailableFailureDomain",
}

var ownerReferenceNotSet = &microerror.Error{
	Kind: "ownerReferenceNotSet",
}

// IsOwnerReferenceNotSet asserts ownerReferenceNotSet.
func IsOwnerReferenceNotSet(err error) bool {
	return microerror.Cause(err) == ownerReferenceNotSet
}
