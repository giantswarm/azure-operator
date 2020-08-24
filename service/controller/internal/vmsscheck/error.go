package vmsscheck

import (
	"strings"

	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

// IsNotFound asserts resource not found error messages from upstream API's response.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(microerror.Cause(err).Error(), "ResourceNotFound") ||
		strings.Contains(microerror.Cause(err).Error(), "ResourceGroupNotFound")
}

var vmssUnsafeError = &microerror.Error{
	Kind: "vmssUnsafeError",
}

// IsVMSSUnsafeError asserts vmssUnsafeError.
func IsVMSSUnsafeError(err error) bool {
	return microerror.Cause(err) == vmssUnsafeError
}
