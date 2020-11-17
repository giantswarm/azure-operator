package conditions

import (
	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
)

func IsUpgradingInProgress(cr CR, desiredRelease string) (bool, error) {
	if IsUpgradingTrue(cr) {
		// When Upgrading == True => Upgrading is still in progress.
		return true, nil
	}

	// If release label isn't updated yet, it means that upgrade hasn't been
	// triggered yet.
	if cr.GetLabels()[label.ReleaseVersion] != desiredRelease {
		return false, nil
	}

	// release.giantswarm.io/version is updated, which means that the node pool
	// upgrade has been started, let's see if it is still in progress.

	lastDeployedReleaseVersion, isSet := cr.GetAnnotations()[annotation.LastDeployedReleaseVersion]
	if !isSet {
		// If release.giantswarm.io/last-deployed-version is not set, that
		// means that the node pool is still being created, so no upgrade in
		// progress.
		return false, nil
	}

	// If last deployed release version is equal to the desired release version
	// while Upgrading condition status is not True, it means that upgrade has
	// been completed already.
	if lastDeployedReleaseVersion == desiredRelease {
		return false, nil
	}

	// At this point we know that CR labels were updated for upgrade but the
	// condition didn't pick up yet. Therefore we consider the upgrading to be
	// in progress.
	return true, nil
}

func IsUpgraded(cr CR, desiredRelease string) (bool, error) {
	if IsUpgradingTrue(cr) {
		// When Upgrading == True => Upgrading is still in progress.
		return false, nil
	}

	lastDeployedReleaseVersion, isSet := cr.GetAnnotations()[annotation.LastDeployedReleaseVersion]
	if !isSet {
		// if release.giantswarm.io/last-deployed-version is not set, that
		// means that the node pool is still being created, so upgrade is not
		// completed.
		return false, nil
	}

	// When CR release label matches desired version & last deployed release
	// version also matches the desired release version, we know that the
	// upgrade has completed.
	if cr.GetLabels()[label.ReleaseVersion] == desiredRelease && lastDeployedReleaseVersion == desiredRelease {
		return true, nil
	}

	return false, nil
}

func MarkUpgradingNotStarted(cr CR) {
	capiconditions.MarkFalse(
		cr,
		aeconditions.UpgradingCondition,
		aeconditions.UpgradeNotStartedReason,
		capi.ConditionSeverityInfo,
		"Upgrade not started")
}

func MarkUpgradingStarted(cr CR) {
	capiconditions.MarkTrue(cr, aeconditions.UpgradingCondition)
}

func MarkUpgradingCompleted(cr CR) {
	capiconditions.MarkFalse(
		cr,
		aeconditions.UpgradingCondition,
		aeconditions.UpgradeCompletedReason,
		capi.ConditionSeverityInfo,
		"Upgrade has been completed")
}
