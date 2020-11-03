package conditions

import (
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func IsUpgradingInProgress(cr CR, desiredRelease string) (bool, error) {
	c := capiconditions.Get(cr, aeconditions.UpgradingCondition)
	if c.Status == corev1.ConditionTrue {
		// When Upgrading == True => Upgrading is still in progress.
		return false, nil
	}

	// If release label isn't updated yet, it means that upgrade hasn't been
	// triggered yet.
	if cr.GetLabels()[label.ReleaseVersion] != desiredRelease {
		return false, nil
	}

	msg, err := aeconditions.DeserializeUpgradingConditionMessage(c.Message)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// If release version in condition metadata contains desired release
	// version while condition status is not True, it means that upgrade has
	// been completed already.
	if msg.ReleaseVersion == desiredRelease {
		return false, nil
	}

	// At this point we know that CR labels were updated for upgrade but the
	// condition didn't pick up yet. Therefore we consider the upgrading to be
	// in progress.
	return true, nil
}

func IsUpgraded(cr CR, desiredRelease string) (bool, error) {
	c := capiconditions.Get(cr, aeconditions.UpgradingCondition)
	if c.Status == corev1.ConditionTrue {
		// When Upgrading == True => Upgrading is still in progress.
		return false, nil
	}

	msg, err := aeconditions.DeserializeUpgradingConditionMessage(c.Message)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// When CR release label matches desired version & upgrading condition's
	// metadata also contains desired release version we know that the upgrade
	// has completed.
	if cr.GetLabels()[label.ReleaseVersion] == desiredRelease && msg.ReleaseVersion == desiredRelease {
		return true, nil
	}

	return false, nil
}

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
