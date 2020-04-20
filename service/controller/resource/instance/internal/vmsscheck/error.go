package vmsscheck

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var vmssUnsafeError = &microerror.Error{
	Kind: "vmssUnsafeError",
}

// IsVMSSUnsafeError asserts vmssUnsafeError.
func IsVMSSUnsafeError(err error) bool {
	return microerror.Cause(err) == vmssUnsafeError
}
