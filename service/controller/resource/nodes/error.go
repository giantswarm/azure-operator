package nodes

import (
	"github.com/giantswarm/microerror"
)

var clientNotFoundError = &microerror.Error{
	Kind: "clientNotFoundError",
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}
