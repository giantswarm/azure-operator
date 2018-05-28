package deployment

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
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
		fmt.Printf("\n")
		fmt.Printf("dErr: %#v\n", dErr)
		fmt.Printf("\n")
		if ok {
			sErr, ok := dErr.Original.(azure.ServiceError)
			fmt.Printf("\n")
			fmt.Printf("sErr: %#v\n", sErr)
			fmt.Printf("\n")
			if ok {
				if sErr.Code == "DeploymentNotFound" {
					return true
				}
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
