package endpoints

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

var incorrectNumberNetworkInterfacesError = &microerror.Error{
	Kind: "incorrectNumberNetworkInterfacesError",
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

var privateIPAddressEmptyError = &microerror.Error{
	Kind: "privateIPAddressEmptyError",
}

var networkInterfacesNotFoundError = &microerror.Error{
	Kind: "networkInterfacesNotFoundError",
}

// IsNetworkInterfacesNotFound asserts networkInterfacesNotFoundError.
func IsNetworkInterfacesNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == networkInterfacesNotFoundError {
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

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
