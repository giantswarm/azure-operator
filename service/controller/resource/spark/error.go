package spark

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

var requirementsNotMetError = &microerror.Error{
	Kind: "requirementsNotMetError",
}

// IsRequirementsNotMet asserts requirementsNotMetError.
func IsRequirementsNotMet(err error) bool {
	return microerror.Cause(err) == requirementsNotMetError
}
