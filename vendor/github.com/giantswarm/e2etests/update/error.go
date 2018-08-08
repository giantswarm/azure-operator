package update

import "github.com/giantswarm/microerror"

var hasDesiredStatusError = &microerror.Error{
	Kind: "hasDesiredStatusError",
}

// IsHasDesiredStatus asserts hasDesiredStatusError.
func IsHasDesiredStatus(err error) bool {
	return microerror.Cause(err) == hasDesiredStatusError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var missesDesiredStatusError = &microerror.Error{
	Kind: "missesDesiredStatusError",
}

// IsMissesDesiredStatus asserts missesDesiredStatusError.
func IsMissesDesiredStatus(err error) bool {
	return microerror.Cause(err) == missesDesiredStatusError
}
