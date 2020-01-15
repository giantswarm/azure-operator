// +build k8srequired

package multiaz

import (
	"github.com/giantswarm/microerror"
)

var wrongAzsUsed = &microerror.Error{
	Desc: "The tenant cluster is not using the specified availability zones.",
	Kind: "wrongAzsUsed",
}

// IsWrongAzsUsed asserts wrongAzsUsed.
func IsWrongAzsUsed(err error) bool {
	return microerror.Cause(err) == wrongAzsUsed
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
