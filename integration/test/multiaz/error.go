// +build k8srequired

package multiaz

import (
	"github.com/giantswarm/microerror"
)

var wrongAvailabilityZones = &microerror.Error{
	Desc: "The tenant cluster is not using the specified availability zones.",
	Kind: "wrongAvailabilityZones",
}

// IsWrongAzsUsed asserts wrongAvailabilityZones.
func IsWrongAvailabilityZones(err error) bool {
	return microerror.Cause(err) == wrongAvailabilityZones
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
