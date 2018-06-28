package key

import "github.com/giantswarm/microerror"

var missingOutputValueError = microerror.New("missing output value")

// IsMissingOutputValue asserts missingOutputValueError.
func IsMissingOutputValue(err error) bool {
	return microerror.Cause(err) == missingOutputValueError
}

var wrongTypeError = microerror.New("wrong type")

// IsWrongTypeError asserts wrongTypeError.
func IsWrongTypeError(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
