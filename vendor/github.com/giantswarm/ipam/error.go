package ipam

import (
	"github.com/giantswarm/microerror"
)

var incorrectNumberOfBoundariesError = &microerror.Error{
	Kind: "incorrectNumberOfBoundariesError",
}

// IsIncorrectNumberOfBoundaries asserts incorrectNumberOfBoundariesError.
func IsIncorrectNumberOfBoundaries(err error) bool {
	return microerror.Cause(err) == incorrectNumberOfBoundariesError
}

var incorrectNumberOfFreeRangesError = &microerror.Error{
	Kind: "incorrectNumberOfFreeRangesError",
}

// IsIncorrectNumberOfFreeRangesError asserts incorrectNumberOfFreeRangesError.
func IsIncorrectNumberOfFreeRanges(err error) bool {
	return microerror.Cause(err) == incorrectNumberOfFreeRangesError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var invalidParameterError = &microerror.Error{
	Kind: "invalid parameter",
}

// IsInvalidParameter asserts invalidParameterError.
func IsInvalidParameter(err error) bool {
	return microerror.Cause(err) == invalidParameterError
}

var ipNotContainedError = &microerror.Error{
	Kind: "ipNotContainedError",
}

// IsIPNotContained asserts ipNotContainedError.
func IsIPNotContained(err error) bool {
	return microerror.Cause(err) == ipNotContainedError
}

var maskIncorrectSizeError = &microerror.Error{
	Kind: "maskIncorrectSizeError",
}

// IsMaskIncorrectSize asserts maskIncorrectSizeError.
func IsMaskIncorrectSize(err error) bool {
	return microerror.Cause(err) == maskIncorrectSizeError
}

var maskTooBigError = &microerror.Error{
	Kind: "maskTooBigError",
}

// IsMaskTooBig asserts maskTooBigError.
func IsMaskTooBig(err error) bool {
	return microerror.Cause(err) == maskTooBigError
}

var nilIPError = &microerror.Error{
	Kind: "nilIPError",
}

// IsNilIP asserts nilIPError.
func IsNilIP(err error) bool {
	return microerror.Cause(err) == nilIPError
}

var spaceExhaustedError = &microerror.Error{
	Kind: "spaceExhaustedError",
}

// IsSpaceExhausted asserts spaceExhaustedError.
func IsSpaceExhausted(err error) bool {
	return microerror.Cause(err) == spaceExhaustedError
}
