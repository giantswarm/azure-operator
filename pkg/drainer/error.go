package drainer

import (
	"strings"

	"github.com/giantswarm/microerror"
)

var alreadyCordonedError = &microerror.Error{
	Kind: "alreadyCordonedError",
}

func IsAlreadyCordoned(err error) bool {
	return microerror.Cause(err) == alreadyCordonedError
}

var cannotEvictPodError = &microerror.Error{
	Kind: "cannotEvictPodError",
}

func IsCannotEvictPod(err error) bool {
	c := microerror.Cause(err)

	if err == nil {
		return false
	}

	if strings.Contains(c.Error(), "Cannot evict pod") {
		return true
	}

	return c == cannotEvictPodError
}

var evictionInProgressError = &microerror.Error{
	Kind: "evictionInProgressError",
}

func IsEvictionInProgress(err error) bool {
	return microerror.Cause(err) == evictionInProgressError
}

var drainTimeoutError = &microerror.Error{
	Kind: "drainTimeoutError",
}

// IsDrainTimeout asserts drainTimeoutError.
func IsDrainTimeout(err error) bool {
	return microerror.Cause(err) == drainTimeoutError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}
