package conditions

import (
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

var unexpectedConditionStatusError = &microerror.Error{
	Kind: "UnexpectedConditionStatus",
}

func UnexpectedConditionStatusError(cr CR, t capi.ConditionType) error {
	c := capiconditions.Get(cr, t)

	return microerror.Maskf(
		unexpectedConditionStatusError,
		"Unexpected status for condition %s, got %s", t, c.Status)
}

// IsInvalidCondition asserts invalidConditionError.
func IsUnexpectedConditionStatus(err error) bool {
	return microerror.Cause(err) == unexpectedConditionStatusError
}
