package conditions

import (
	"fmt"

	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

var UnexpectedConditionStatusError = &microerror.Error{
	Kind: "UnexpectedConditionStatus",
}

func UnexpectedConditionStatusErrorMessage(cr CR, t capi.ConditionType) string {
	c := capiconditions.Get(cr, t)
	return fmt.Sprintf("Unexpected status for condition %s, got %s", t, c.Status)
}

// IsInvalidCondition asserts invalidConditionError.
func IsUnexpectedConditionStatus(err error) bool {
	return microerror.Cause(err) == UnexpectedConditionStatusError
}
