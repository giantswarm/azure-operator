package senddecorator

import "github.com/giantswarm/microerror"

var tooManyRequestsError = &microerror.Error{
	Kind: "tooManyRequestsError",
}

// IsTooManyRequests asserts tooManyRequestsError.
func IsTooManyRequests(err error) bool {
	return microerror.Cause(err) == tooManyRequestsError
}
