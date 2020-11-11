package conditions

import (
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

func IsTrue(condition *capi.Condition) bool {
	return condition != nil && condition.Status == corev1.ConditionTrue
}

func IsFalse(condition *capi.Condition) bool {
	return condition != nil && condition.Status == corev1.ConditionFalse
}

func IsUnknown(condition *capi.Condition) bool {
	return condition == nil || condition.Status == corev1.ConditionUnknown
}

func IsUpgradingTrue(from capiconditions.Getter) bool {
	return capiconditions.IsTrue(from, aeconditions.UpgradingCondition)
}

func IsUpgradingFalse(from capiconditions.Getter) bool {
	return capiconditions.IsFalse(from, aeconditions.UpgradingCondition)
}

func IsUnexpected(from capiconditions.Getter, t capi.ConditionType) bool {
	condition := capiconditions.Get(from, t)

	// We expect that condition is not set and that case should be handled
	// separately where necessary.
	if condition == nil {
		return false
	}

	switch condition.Status {
	// We expect currently known upstream values: True, False and Unknown.
	case corev1.ConditionTrue, corev1.ConditionFalse, corev1.ConditionUnknown:
		return false
	// Everything else is not supported.
	default:
		return true
	}
}
