package credential

import (
	"github.com/giantswarm/microerror"
)

var missingValueError = &microerror.Error{
	Kind: "missingValueError",
}
