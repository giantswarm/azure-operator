package deployment

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

var timeoutError = microerror.New("timeout")

// IsTimeoutError asserts createTimeoutError.
func IsTimeoutError(err error) bool {
	return microerror.Cause(err) == timeoutError
}

var invalidConfigError = microerror.New("invalid config")

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var notFoundError = microerror.New("not found")

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == notFoundError {
		return true
	}

	fmt.Printf("\n")
	fmt.Printf("c: %#v\n", c)
	fmt.Printf("\n")

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

var wrongTypeError = microerror.New("wrong type")

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
