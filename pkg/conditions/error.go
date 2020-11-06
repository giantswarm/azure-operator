package conditions

import (
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
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

func ExpectedTrueErrorMessage(cr CR, conditionType capi.ConditionType) string {
	return ExpectedStatusErrorMessage(cr, conditionType, corev1.ConditionTrue)
}

func ExpectedFalseErrorMessage(cr CR, conditionType capi.ConditionType) string {
	return ExpectedStatusErrorMessage(cr, conditionType, corev1.ConditionFalse)
}

func ExpectedStatusErrorMessage(cr CR, conditionType capi.ConditionType, expectedStatus corev1.ConditionStatus) string {
	c := capiconditions.Get(cr, conditionType)
	var got string
	if c != nil {
		got = string(c.Status)
	} else {
		got = "condition not set"
	}

	return fmt.Sprintf("Expected that condition %s on CR %T has status %s, but got %s", conditionType, cr, expectedStatus, got)
}

// IsInvalidCondition asserts invalidConditionError.
func IsUnexpectedConditionStatus(err error) bool {
	return microerror.Cause(err) == UnexpectedConditionStatusError
}
