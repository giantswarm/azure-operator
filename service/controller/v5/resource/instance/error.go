package instance

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

var executionFailedError = &microerror.Error{
	Kind: "executionFailedError",
}

// IsExecutionFailed asserts executionFailedError.
func IsExecutionFailed(err error) bool {
	return microerror.Cause(err) == executionFailedError
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
