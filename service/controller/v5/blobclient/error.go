package blobclient

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
	"strings"
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

// IsBlobNotFound asserts blob not found error from upstream's API message.
func IsBlobNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(microerror.Cause(err).Error(), "ServiceCode=BlobNotFound")
}

// IsContainerNotFound asserts container not found error from upstream's API message.
func IsContainerNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(microerror.Cause(err).Error(), "ContainerNotFound")
}

// IsStorageAccountNotFound asserts storage account not found error from upstream's API message.
func IsStorageAccountNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(microerror.Cause(err).Error(), "ResourceNotFound")
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
