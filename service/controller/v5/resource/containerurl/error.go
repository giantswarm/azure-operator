package containerurl

import (
	"strings"

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

// IsStorageAccountNotFound asserts storage account not found error from upstream's API message.
func IsStorageAccountNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(microerror.Cause(err).Error(), "ResourceNotFound")
}
