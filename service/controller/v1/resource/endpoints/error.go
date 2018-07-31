package endpoints

import "github.com/giantswarm/microerror"

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

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}

var incorrectNumberNetworkInterfacesError = &microerror.Error{
	Kind: "incorrectNumberNetworkInterfacesError",
}

// IsincorrectNumberNetworkInterfacesError asserts incorrectNumberNetworkInterfacesError.
func IsIncorrectNumberNetworkInterfacesError(err error) bool {
	return microerror.Cause(err) == incorrectNumberNetworkInterfacesError
}

var privateIPAddressEmptyError = &microerror.Error{
	Kind: "privateIPAddressEmptyError",
}

// IsPrivateIPAddressEmptyError asserts privateIPAddressEmptyError.
func IsPrivateIPAddressEmptyError(err error) bool {
	return microerror.Cause(err) == privateIPAddressEmptyError
}
