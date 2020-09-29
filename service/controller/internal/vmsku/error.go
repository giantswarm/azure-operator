package vmsku

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var skuNotFoundError = &microerror.Error{
	Kind: "skuNotFoundError",
}

// IsSkuNotFoundError asserts skuNotFoundError.
func IsSkuNotFoundError(err error) bool {
	return microerror.Cause(err) == skuNotFoundError
}
