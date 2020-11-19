package subnet

import (
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfigError asserts invalidConfigError.
func IsInvalidConfigError(err error) bool {
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

	return strings.Contains(microerror.Cause(err).Error(), "ResourceNotFound") ||
		strings.Contains(microerror.Cause(err).Error(), "ResourceGroupNotFound")
}

var missingOutputValueError = &microerror.Error{
	Kind: "missingOutputValueError",
}

// IsMissingOutputValue asserts missingOutputValueError.
func IsMissingOutputValue(err error) bool {
	return microerror.Cause(err) == missingOutputValueError
}

var tooManyCredentialsError = &microerror.Error{
	Kind: "tooManyCredentialsError",
}

// IsTooManyCredentialsError asserts tooManyCredentialsError.
func IsTooManyCredentialsError(err error) bool {
	return microerror.Cause(err) == tooManyCredentialsError
}

var missingOrganizationLabel = &microerror.Error{
	Kind: "missingOrganizationLabel",
}

// IsMissingOrganizationLabel asserts missingOrganizationLabel.
func IsMissingOrganizationLabel(err error) bool {
	return microerror.Cause(err) == missingOrganizationLabel
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}

// IsSubnetInUse asserts subnet in use error from Azure API message.
func IsSubnetInUse(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(microerror.Cause(err).Error(), "is in use by")
}

var natGatewayNotReadyError = &microerror.Error{
	Kind: "natGatewayNotReadyError",
}

// IsNatGatewayNotReadyError asserts natGatewayNotReadyError.
func IsNatGatewayNotReadyError(err error) bool {
	return microerror.Cause(err) == natGatewayNotReadyError
}

// IsStorageAccountNotFound asserts storage account not found error from Azure API message.
func IsStorageAccountNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(microerror.Cause(err).Error(), "StorageAccountNotFound")
}
