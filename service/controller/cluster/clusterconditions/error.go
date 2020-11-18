package clusterconditions

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var invalidConditionError = &microerror.Error{
	Kind: "invalidConditionError",
}

// IsInvalidCondition asserts invalidConditionError.
func IsInvalidCondition(err error) bool {
	return microerror.Cause(err) == invalidConditionError
}
