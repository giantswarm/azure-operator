package endpoints

import "github.com/giantswarm/microerror"

var invalidConfigError = microerror.New("invalid config")

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var notFoundError = microerror.New("not found")

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}

var wrongTypeError = microerror.New("wrong type")

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}

var incorrectNumberNetworkInterfacesError = microerror.New("incorrect number network interfaces")

// IsincorrectNumberNetworkInterfacesError asserts incorrectNumberNetworkInterfacesError.
func IsIncorrectNumberNetworkInterfacesError(err error) bool {
	return microerror.Cause(err) == incorrectNumberNetworkInterfacesError
}

var privateIPAddressEmptyError = microerror.New("private ip address empty")

// IsPrivateIPAddressEmptyError asserts privateIPAddressEmptyError.
func IsPrivateIPAddressEmptyError(err error) bool {
	return microerror.Cause(err) == privateIPAddressEmptyError
}
