package credential

import (
	"github.com/giantswarm/microerror"
)

var azureClusterNotFoundError = &microerror.Error{
	Kind: "azureClusterNotFoundError",
}

var identityRefNotSetError = &microerror.Error{
	Kind: "identityRefNotSetError",
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var missingValueError = &microerror.Error{
	Kind: "missingValueError",
}

var subscriptionIDNotSetError = &microerror.Error{
	Kind: "subscriptionIDNotSetError",
}
