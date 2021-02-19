package credentialprovider

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

var azureClusterNotFoundError = &microerror.Error{
	Kind: "azureClusterNotFoundError",
}

var identityRefNotSetError = &microerror.Error{
	Kind: "identityRefNotSetError",
}

var subscriptionIDNotSetError = &microerror.Error{
	Kind: "subscriptionIDNotSetError",
}

var missingValueError = &microerror.Error{
	Kind: "missingValueError",
}

var credentialsNotFoundError = &microerror.Error{
	Kind: "credentialsNotFoundError",
}

// IsCredentialsNotFoundError asserts credentialsNotFoundError.
func IsCredentialsNotFoundError(err error) bool {
	return microerror.Cause(err) == credentialsNotFoundError
}

var tooManyCredentialsError = &microerror.Error{
	Kind: "tooManyCredentialsError",
}

// IsTooManyCredentials asserts tooManyCredentialsError.
func IsTooManyCredentials(err error) bool {
	return microerror.Cause(err) == tooManyCredentialsError
}

var notImplementedError = &microerror.Error{
	Kind: "notImplementedError",
}
