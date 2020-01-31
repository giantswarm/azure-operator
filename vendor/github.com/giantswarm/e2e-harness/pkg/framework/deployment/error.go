package deployment

import "github.com/giantswarm/microerror"

var incorrectDeploymentError = &microerror.Error{
	Kind: "incorrectDeploymentError",
}

// IsIncorrectDeployment asserts incorrectDeploymentError.
func IsIncorrectDeployment(err error) bool {
	return microerror.Cause(err) == incorrectDeploymentError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}
