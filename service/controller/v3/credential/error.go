package credential

import "github.com/giantswarm/microerror"

var invalidConfig = &microerror.Error{
	Kind: "invalidConfig",
}

// IsInvalidConfigFoundError asserts invalidConfig.
func IsInvalidConfigFoundError(err error) bool {
	return microerror.Cause(err) == invalidConfig
}
