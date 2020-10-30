package conditions

import (
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func MarkUpgradingNotStarted(cr CR) error {
	// Cluster is just being created, no upgrade yet.
	message := aeconditions.UpgradingConditionMessage{
		Message:        "Upgrade not started",
		ReleaseVersion: key.ReleaseVersion(cr),
	}
	messageString, err := aeconditions.SerializeUpgradingConditionMessage(message)
	if err != nil {
		return microerror.Mask(err)
	}

	capiconditions.MarkFalse(
		cr,
		aeconditions.UpgradingCondition,
		aeconditions.UpgradeNotStartedReason,
		capi.ConditionSeverityInfo,
		messageString)

	return nil
}

func MarkUpgradingStarted(cr CR) {
	capiconditions.MarkTrue(cr, aeconditions.UpgradingCondition)
}

func MarkUpgradingCompleted(cr CR) error {
	message := aeconditions.UpgradingConditionMessage{
		Message:        "Upgrade has been completed",
		ReleaseVersion: key.ReleaseVersion(cr),
	}
	messageString, err := aeconditions.SerializeUpgradingConditionMessage(message)
	if err != nil {
		return microerror.Mask(err)
	}

	capiconditions.MarkFalse(
		cr,
		aeconditions.UpgradingCondition,
		aeconditions.UpgradeCompletedReason,
		capi.ConditionSeverityInfo,
		messageString)

	return nil
}
