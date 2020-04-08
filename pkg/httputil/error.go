package httputil

import "github.com/giantswarm/microerror"

var parseError = &microerror.Error{
	Kind: "parseError",
}

// IsParse asserts parseError.
func IsParse(err error) bool {
	return microerror.Cause(err) == parseError
}
