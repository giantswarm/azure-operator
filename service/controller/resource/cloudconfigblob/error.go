package cloudconfigblob

import (
	"strings"

	"github.com/giantswarm/microerror"
)

// executionFailedError is an error type for situations where Resource
// execution cannot continue and must always fall back to operatorkit.
//
// This error should never be matched against and therefore there is no matcher
// implement. For further information see:
//
//     https://github.com/giantswarm/fmt/blob/master/go/errors.md#matching-errors
//
var executionFailedError = &microerror.Error{
	Kind: "executionFailedError",
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

// IsStorageAccountNotFound asserts storage account not found error from upstream's API message.
func IsStorageAccountNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(microerror.Cause(err).Error(), "ResourceNotFound") ||
		strings.Contains(microerror.Cause(err).Error(), "StorageAccountNotFound")
}

// IsStorageAccountNotProvisioned asserts storage account not provisioned error from upstream's API message.
func IsStorageAccountNotProvisioned(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(microerror.Cause(err).Error(), "StorageAccountIsNotProvisioned")
}

var tooManyCredentialsError = &microerror.Error{
	Kind: "tooManyCredentialsError",
}

// IsTooManyCredentialsError asserts tooManyCredentialsError.
func IsTooManyCredentialsError(err error) bool {
	return microerror.Cause(err) == tooManyCredentialsError
}

var missingOrganizationLabel = &microerror.Error{
	Kind: "missingOrganizationLabel",
}

// IsMissingOrganizationLabel asserts missingOrganizationLabel.
func IsMissingOrganizationLabel(err error) bool {
	return microerror.Cause(err) == missingOrganizationLabel
}

var bootstrapCRNotReady = &microerror.Error{
	Kind: "bootstrapCRNotReady",
}

// IsMissingCloudConfigSecret asserts bootstrapCRNotReady.
func IsBootstrapCRNotReady(err error) bool {
	return microerror.Cause(err) == bootstrapCRNotReady
}

var ownerReferenceNotSet = &microerror.Error{
	Kind: "ownerReferenceNotSet",
}

// IsOwnerReferenceNotSet asserts ownerReferenceNotSet.
func IsOwnerReferenceNotSet(err error) bool {
	return microerror.Cause(err) == ownerReferenceNotSet
}
