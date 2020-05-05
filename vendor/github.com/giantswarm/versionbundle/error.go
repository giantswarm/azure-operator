package versionbundle

import (
	"github.com/giantswarm/microerror"
)

var bundleNotFoundError = &microerror.Error{
	Kind: "bundleNotFoundError",
}

// IsBundleNotFound asserts bundleNotFoundError.
func IsBundleNotFound(err error) bool {
	return microerror.Cause(err) == bundleNotFoundError
}

var executionFailedError = &microerror.Error{
	Kind: "executionFailedError",
}

// IsExecutionFailed asserts executionFailedError.
func IsExecutionFailed(err error) bool {
	return microerror.Cause(err) == executionFailedError
}

var invalidBundleError = &microerror.Error{
	Kind: "invalidBundleError",
}

// IsInvalidBundle asserts invalidBundleError.
func IsInvalidBundle(err error) bool {
	return microerror.Cause(err) == invalidBundleError
}

// IsInvalidBundleError asserts invalidBundleError.
func IsInvalidBundleError(err error) bool {
	return microerror.Cause(err) == invalidBundleError
}

var invalidBundlesError = &microerror.Error{
	Kind: "invalidBundlesError",
}

// IsInvalidBundles asserts invalidBundlesError.
func IsInvalidBundles(err error) bool {
	return microerror.Cause(err) == invalidBundlesError
}

// IsInvalidBundlesError asserts invalidBundlesError.
func IsInvalidBundlesError(err error) bool {
	return microerror.Cause(err) == invalidBundlesError
}

var invalidChangelogError = &microerror.Error{
	Kind: "invalidChangelogError",
}

// IsInvalidChangelog asserts invalidChangelogError.
func IsInvalidChangelog(err error) bool {
	return microerror.Cause(err) == invalidChangelogError
}

var invalidComponentError = &microerror.Error{
	Kind: "invalidComponentError",
}

// IsInvalidComponent asserts invalidComponentError.
func IsInvalidComponent(err error) bool {
	return microerror.Cause(err) == invalidComponentError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var invalidReleaseError = &microerror.Error{
	Kind: "invalidReleaseError",
}

// IsInvalidRelease asserts invalidReleaseError.
func IsInvalidRelease(err error) bool {
	return microerror.Cause(err) == invalidReleaseError
}
