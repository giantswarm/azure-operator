package nodes

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

var clientNotFoundError = &microerror.Error{
	Kind: "clientNotFoundError",
}

// IsClientNotFound asserts clientNotFoundError.
func IsClientNotFound(err error) bool {
	return microerror.Cause(err) == clientNotFoundError
}

// executionFailedError is an error type for situations where Resource
// execution cannot continue and must always fall back to operatorkit.
//
// This error should never be matched against and therefore there is no matcher
// implement. For further information see:
//
//	https://github.com/giantswarm/fmt/blob/master/go/errors.md#matching-errors
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

var scaleSetNotFoundError = &microerror.Error{
	Kind: "scaleSetNotFoundError",
}

// IsScaleSetNotFound asserts scaleSetNotFoundError.
func IsScaleSetNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == scaleSetNotFoundError {
		return true
	}

	{
		dErr, ok := c.(autorest.DetailedError)
		if ok {
			if dErr.StatusCode == 404 {
				return true
			}
		}
	}

	return false
}
