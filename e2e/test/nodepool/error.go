// +build k8srequired

package nodepool

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

var unexpectedNumberOfNodesError = &microerror.Error{
	Kind: "unexpectedNumberOfNodesError",
}

// IsUnexpectedNumberOfNodesError asserts unexpectedNumberOfNodesError.
func IsUnexpectedNumberOfNodesError(err error) bool {
	return microerror.Cause(err) == unexpectedNumberOfNodesError
}

var missingNodePoolLabelError = &microerror.Error{
	Kind: "missingNodePoolLabelError",
}

// IsMissingNodePoolLabelError asserts missingNodePoolLabelError.
func IsMissingNodePoolLabelError(err error) bool {
	return microerror.Cause(err) == missingNodePoolLabelError
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFoundError asserts notFoundError.
func IsNotFoundError(err error) bool {
	return microerror.Cause(err) == notFoundError
}

var sameVmSizeError = &microerror.Error{
	Kind: "sameVmSizeError",
}

// IsSameVmSizeError asserts sameVmSizeError.
func IsSameVmSizeError(err error) bool {
	return microerror.Cause(err) == sameVmSizeError
}

var waitError = &microerror.Error{
	Kind: "waitError",
}

// IsWait asserts waitError.
func IsWait(err error) bool {
	return microerror.Cause(err) == waitError
}
