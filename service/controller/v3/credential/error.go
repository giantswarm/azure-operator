package credential

import "github.com/giantswarm/microerror"

var invalidConfig = microerror.New("invalid config")

// IsInvalidConfigFoundError asserts invalidConfig.
func IsInvalidConfigFoundError(err error) bool {
	return microerror.Cause(err) == invalidConfig
}
