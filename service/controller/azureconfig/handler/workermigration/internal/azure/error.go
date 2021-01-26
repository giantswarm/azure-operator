package azure

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

// IsNotFound asserts generic Azure API not found error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

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
