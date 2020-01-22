package instance

import (
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

var missingLabelError = &microerror.Error{
	Kind: "missingLabelError",
}

// IsMissingLabel asserts missingLabelError.
func IsMissingLabel(err error) bool {
	return microerror.Cause(err) == missingLabelError
}

var scaleSetNotFoundError = &microerror.Error{
	Kind: "scaleSetNotFoundError",
}

// IsScaleSetNotFound asserts scaleSetNotFoundError.
func IsScaleSetNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == scaleSetNotFoundError {
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

var versionBlobEmptyError = &microerror.Error{
	Kind: "versionBlobEmptyError",
}

// IsVersionBlobEmpty asserts versionBlobEmptyError.
func IsVersionBlobEmpty(err error) bool {
	return microerror.Cause(err) == versionBlobEmptyError
}

var nilTemplateLinkError = &microerror.Error{
	Kind: "nilTemplateLink",
}

func IsNilTemplateLinkError(err error) bool {
	return microerror.Cause(err) == nilTemplateLinkError
}

var unableToGetTemplateError = &microerror.Error{
	Kind: "unableToGetTemplate",
}

func IsUnableToGetTemplateError(err error) bool {
	return microerror.Cause(err) == unableToGetTemplateError
}
