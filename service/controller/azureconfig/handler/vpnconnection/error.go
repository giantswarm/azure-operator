package vpnconnection

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}

var vpnGatewayNotFoundError = &microerror.Error{
	Kind: "vpnGatewayNotFoundError",
}

// IsVPNGatewayNotFound asserts vpnGatewayNotFoundError.
func IsVPNGatewayNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == vpnGatewayNotFoundError {
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

var vpnGatewayConnectionNotFoundError = &microerror.Error{
	Kind: "vpnGatewayConnectionNotFoundError",
}

// IsVPNGatewayConnectionNotFound asserts vpnGatewayConnectionNotFoundError.
func IsVPNGatewayConnectionNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == vpnGatewayConnectionNotFoundError {
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
