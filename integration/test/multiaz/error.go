// +build k8srequired

package multiaz

import (
	"github.com/giantswarm/microerror"
)

var wrongAZzUsed = &microerror.Error{
	Desc: "The tenant cluster is not using the specified availability zones.",
	Kind: "wrongAZzUsed",
}

// IsWrongAZsUsed asserts wrongAZzUsed.
func IsWrongAZsUsed(err error) bool {
	return microerror.Cause(err) == wrongAZzUsed
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
