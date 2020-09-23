package nodestatus

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfigError asserts invalidConfigError.
func IsInvalidConfigError(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var errNoAvailableNodes = &microerror.Error{
	Kind: "errNoAvailableNodes",
}

// IsErrNoAvailableNodes asserts errNoAvailableNodes.
func IsErrNoAvailableNodes(err error) bool {
	return microerror.Cause(err) == errNoAvailableNodes
}
