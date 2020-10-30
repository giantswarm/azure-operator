package clusterconditions

import (
	"context"
	"time"

	"github.com/blang/semver"
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/conditions"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) ensureUpgradingCondition(ctx context.Context, cluster *capi.Cluster) error {
	// Case 1: new cluster just being created, no upgrade yet.
	if capiconditions.IsTrue(cluster, aeconditions.CreatingCondition) {
		err := conditions.MarkUpgradingNotStarted(cluster)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	// Case 2: Upgrading=False, cluster is currently not in upgrading state,
	// let's check if it should be.
	if capiconditions.IsFalse(cluster, aeconditions.UpgradingCondition) {
		err := r.checkIfUpgradingHasBeenStarted(ctx, cluster)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	// Case 3: Upgrading=True, upgrading is currently in progress, here we
	// check if it has been completed.
	if capiconditions.IsTrue(cluster, aeconditions.UpgradingCondition) {
		err := r.checkInProgressUpgrading(ctx, cluster)
		if err != nil {
			return microerror.Mask(err)
		}

		// Cluster still not Ready, Upgrading remains to be true.
		return nil
	}

	// Case 4: Upgrading condition is not known, which cannot be possible, as
	// API must set it when starting the upgrade.
	if capiconditions.IsUnknown(cluster, aeconditions.UpgradingCondition) {
		return microerror.Maskf(invalidConditionError, "expected that Cluster Upgrading condition is True or False, got Unknown or not set at all")
	}

	// Case 5: We processed Upgrading condition for status values True, False
	// and Unknown (as well as nil for Upgrading condition), so if we got this
	// far in the function, it means that here we have a new condition status
	// value that we do not support and should not be set.
	upgradingCondition := capiconditions.Get(cluster, aeconditions.UpgradingCondition)
	return microerror.Maskf(invalidConditionError, "unexpected Cluster Upgrading condition status %s", upgradingCondition.Status)
}

func (r *Resource) checkIfUpgradingHasBeenStarted(ctx context.Context, cluster *capi.Cluster) error {
	// (1) Cluster is currently not in the Upgrading state (we checked that before getting here).
	upgradingCondition := capiconditions.Get(cluster, aeconditions.UpgradingCondition)

	clusterUpgradingReasonSet :=
		upgradingCondition.Reason == aeconditions.UpgradeNotStartedReason ||
			upgradingCondition.Reason == aeconditions.UpgradeCompletedReason

	// Let's try to get the current release from the last successful upgrade we
	// did.
	if clusterUpgradingReasonSet {
		// (2) This cluster has already been created or upgraded successfully
		// to some release, let's check what was the latest version.
		message, err := aeconditions.DeserializeUpgradingConditionMessage(upgradingCondition.Message)
		if err != nil {
			return microerror.Mask(err)
		}

		latestReleaseVersion, err := semver.ParseTolerant(message.ReleaseVersion)
		if err != nil {
			return microerror.Mask(err)
		}
		desiredReleaseVersion, err := semver.ParseTolerant(key.ReleaseVersion(cluster))
		if err != nil {
			return microerror.Mask(err)
		}

		if desiredReleaseVersion.GT(latestReleaseVersion) {
			// (3) Desired release for this cluster is newer than the release
			// to which it was previously upgraded (or with which was created).
			//
			// Based on (1), (2) and (3), we can conclude that the cluster
			// should be in the Upgrading state.
			conditions.MarkUpgradingStarted(cluster)
			return nil
		}
	}

	// Upgrade has not been started
	err := conditions.MarkUpgradingNotStarted(cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) checkInProgressUpgrading(ctx context.Context, cluster *capi.Cluster) error {
	upgradingCondition := capiconditions.Get(cluster, aeconditions.UpgradingCondition)

	// We expect that the Upgrading is in progress here.
	if upgradingCondition == nil {
		return microerror.Maskf(invalidConditionError, "expected that Cluster Upgrading condition is True, but the condition is nil")
	} else if upgradingCondition.Status != corev1.ConditionTrue {
		return microerror.Maskf(invalidConditionError, "expected that Cluster Upgrading condition is True, got %s", upgradingCondition.Status)
	}

	// Upgrading is in progress, now let's check if it has been completed.

	// But don't check if Upgrading has been completed for the first 5 minutes,
	// give other controllers time to start reconciling their CRs.
	if time.Now().Before(upgradingCondition.LastTransitionTime.Add(5 * time.Minute)) {
		return nil
	}

	// Cluster has been in Upgrading state for at least 5 minutes now, so
	// let's check if it is Ready.
	readyCondition := capiconditions.Get(cluster, capi.ReadyCondition)
	clusterIsReady := readyCondition != nil && readyCondition.Status == corev1.ConditionTrue

	if !clusterIsReady {
		// Cluster still not Ready, Upgrading remains to be true.
		return nil
	}

	// (1) In addition to cluster being ready, here we check that it actually
	// became ready during the upgrade, which would mean that the upgrade has
	// been completed.
	becameReadyWhileUpgrading := readyCondition.LastTransitionTime.After(upgradingCondition.LastTransitionTime.Time)

	// (2) Or we declare Upgrading to be completed if nothing happened for 15
	// minutes, which could currently happen if we were upgrading some
	// component which is not covered by any Ready status condition.
	const upgradingWithoutReadyUpdateThreshold = 15 * time.Minute
	upgradingWithoutReadyUpdateThresholdReached := clusterIsReady && time.Now().After(upgradingCondition.LastTransitionTime.Add(upgradingWithoutReadyUpdateThreshold))

	if becameReadyWhileUpgrading || upgradingWithoutReadyUpdateThresholdReached {
		// Cluster is ready, and either (1) or (2) is true, so we mark upgrade
		// as completed.
		err := conditions.MarkUpgradingCompleted(cluster)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	// Cluster is Ready, but we wait more before marking the upgrade as
	// completed, since neither (1) nor (2) were satisfied.
	return nil
}
