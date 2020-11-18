package deployment

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

var timeoutError = &microerror.Error{
	Kind: "timeoutError",
}

// IsTimeoutError asserts createTimeoutError.
func IsTimeoutError(err error) bool {
	return microerror.Cause(err) == timeoutError
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
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == notFoundError {
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

var missingOutputValueError = &microerror.Error{
	Kind: "missingOutputValueError",
}

// IsMissingOutputValue asserts missingOutputValueError.
func IsMissingOutputValue(err error) bool {
	return microerror.Cause(err) == missingOutputValueError
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}

var nilTemplateLinkError = &microerror.Error{
	Kind: "nilTemplateLink",
}

func IsNilTemplateLinkError(err error) bool {
	return microerror.Cause(err) == nilTemplateLinkError
}

var unableToGetTemplateError = &microerror.Error{
	Kind: "unableToGetTemplate",
}

func IsUnableToGetTemplateError(err error) bool {
	return microerror.Cause(err) == unableToGetTemplateError
}
