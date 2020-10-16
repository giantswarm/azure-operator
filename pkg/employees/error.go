package employees

import (
	"github.com/giantswarm/microerror"
)

var parsingFailedError = &microerror.Error{
	Kind: "parsingFailedError",
}

// IsParsingFailedError asserts parsingFailedError.
func IsParsingFailedError(err error) bool {
	return microerror.Cause(err) == parsingFailedError
}
