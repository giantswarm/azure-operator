package vmsku

import "github.com/giantswarm/microerror"

var skuNotFoundError = &microerror.Error{
	Kind: "skuNotFoundError",
}

// IsSkuNotFoundError asserts skuNotFoundError.
func IsSkuNotFoundError(err error) bool {
	return microerror.Cause(err) == skuNotFoundError
}
