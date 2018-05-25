package resourcegroup

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/giantswarm/microerror"
)

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

	fmt.Printf("err: %#v\n", err)

	c := microerror.Cause(err)

	fmt.Printf("c: %#v\n", c)

	if c == notFoundError {
		return true
	}

	{
		rErr, ok := c.(azure.RequestError)
		if ok {
			fmt.Printf("rErr: %#v\n", rErr)
			fmt.Printf("rErr.ServiceError: %#v\n", rErr.ServiceError)
			if rErr.ServiceError.Code == "404" {
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
