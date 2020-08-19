package ipam

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
)

var parentNetworkRangeStillNotKnown = &microerror.Error{
	Kind: "parentNetworkRangeStillNotKnown",
}

// IsParentNetworkRangeStillNotKnown asserts parentNetworkRangeStillNotKnown. This error is
// returned when allocating node pool subnet, but tenant cluster VNet range is still not known.
func IsParentNetworkRangeStillNotKnown(err error) bool {
	return microerror.Cause(err) == parentNetworkRangeStillNotKnown
}

var invalidConfigError = &microerror.Error{
	Kind: "invalid config",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var invalidObjectError = &microerror.Error{
	Kind: "invalid object",
}

// IsInvalidObject asserts invalidObjectError.
func IsInvalidObject(err error) bool {
	return microerror.Cause(err) == invalidObjectError
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	{
		c := microerror.Cause(err)
		dErr, ok := c.(autorest.DetailedError)
		if ok {
			if dErr.StatusCode == 404 {
				return true
			}
		}
	}

	return false
}
