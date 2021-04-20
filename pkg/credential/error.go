package credential

import (
	"github.com/giantswarm/microerror"
)

var missingValueError = &microerror.Error{
	Kind: "missingValueError",
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var oldStyleCredentialsError = &microerror.Error{
	Kind: "oldStyleCredentialsError",
}
