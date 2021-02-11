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

var emptySubscriptionIDError = &microerror.Error{
	Kind: "emptySubscriptionIDError",
}

var identityRefNotSetError = &microerror.Error{
	Kind: "identityRefNotSetError",
}
