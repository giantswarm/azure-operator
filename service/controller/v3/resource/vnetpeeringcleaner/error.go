package vnetpeeringcleaner

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/giantswarm/microerror"
)

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
		dErr, ok := err.(autorest.DetailedError)
		if ok {
			rErr, ok := (dErr.Original).(*azure.RequestError)
			if ok {
				if rErr.ServiceError.Code == "ResourceNotFound" {
					return true
				}
			}
		}
	}

	return false
}
