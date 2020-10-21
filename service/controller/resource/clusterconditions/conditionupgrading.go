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

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	ClusterCreationInProgressReason = "ClusterCreationInProgress"
	UpgradeCompletedReason          = "UpgradeCompleted"
	UpgradeNotStartedReason         = "UpgradeNotStarted"
	UpgradeCompletedMessagePrefix   = "Successfully upgraded to release "
	UpgradeCompletedMessageFormat   = UpgradeCompletedMessagePrefix + "%s"
)

type CreatingConditionMessage struct {
	Message        string `json:"message"`
	ReleaseVersion string `json:"release_version"`
}

type UpgradingConditionMessage struct {
	Message        string `json:"message"`
	ReleaseVersion string `json:"release_version"`
}

func SerializeUpgradingConditionMessage(message UpgradingConditionMessage) string {
	return ""
}

func DeserializeUpgradingConditionMessage(message string) UpgradingConditionMessage {
	return UpgradingConditionMessage{}
}

func (r *Resource) ensureUpgradingCondition(ctx context.Context, cluster *capi.Cluster) error {
	var err error

	if capiconditions.IsTrue(cluster, aeconditions.CreatingCondition) {
		// Cluster is just being created, no upgrade yet.
		setUpgradeNotStarted(cluster)
		return nil
	}

	upgradingCondition := capiconditions.Get(cluster, aeconditions.UpgradingCondition)

	// Upgrading is in progress, here we check if it has been completed.
	if upgradingCondition != nil && upgradingCondition.Status == corev1.ConditionTrue {
		// Don't check if Upgrading has been completed for the first 5 minutes,
		// give other controllers time to start reconciling their CRs.
		if time.Now().Before(upgradingCondition.LastTransitionTime.Add(5 * time.Minute)) {
			return nil
		}

		// Cluster has been in Upgrading state for at least 5 minutes now, so
		// let's check if it is Ready.
		readyCondition := capiconditions.Get(cluster, capi.ReadyCondition)
		clusterIsReady := readyCondition != nil && readyCondition.Status == corev1.ConditionTrue

		// In addition to cluster being ready, here we check that it actually
		// became ready during the upgrade, which would mean that the upgrade
		// has been completed.
		becameReadyWhileUpgrading := clusterIsReady && readyCondition.LastTransitionTime.After(upgradingCondition.LastTransitionTime.Time)

		// Or we declare Upgrading to be completed if nothing happened for 15
		// minutes, which could currently happen if we were upgrading some
		// component which is not covered by any Ready status condition.
		const upgradingWithoutReadyUpdateThreshold = 15 * time.Minute
		upgradingWithoutReadyUpdateThresholdReached := clusterIsReady && time.Now().After(upgradingCondition.LastTransitionTime.Add(upgradingWithoutReadyUpdateThreshold))

		if becameReadyWhileUpgrading || upgradingWithoutReadyUpdateThresholdReached {
			// Cluster was in Upgrading state, but now it's ready, upgrade has
			// been completed.
			message := UpgradingConditionMessage{
				Message:        "Upgrade has been completed",
				ReleaseVersion: key.ReleaseVersion(cluster),
			}
			messageString := SerializeUpgradingConditionMessage(message)
			capiconditions.MarkFalse(
				cluster,
				aeconditions.UpgradingCondition,
				UpgradeCompletedReason,
				capi.ConditionSeverityInfo,
				messageString)
			return nil
		}

		// Cluster still not Ready, Upgrading remains to be true.
		return nil
	}

	// Here we expect that the Cluster Upgrading state is False
	if !capiconditions.IsFalse(cluster, aeconditions.UpgradingCondition) {
		return microerror.Maskf(invalidConditionError, "expected that Cluster Upgrading condition is False, got %s", upgradingCondition.Status)
	}

	// Cluster is currently not in upgrading state, let's check if it should
	// be.
	isUpgrading, err := r.checkIfUpgradingHasBeenStarted(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	if isUpgrading {
		capiconditions.MarkTrue(cluster, aeconditions.UpgradingCondition)
	} else {
		setUpgradeNotStarted(cluster)
	}

	return nil
}

func (r *Resource) checkIfUpgradingHasBeenStarted(ctx context.Context, cluster *capi.Cluster) (bool, error) {
	// (1) Cluster is currently not in the Creating state (we checked that before getting here).
	upgradingCondition := capiconditions.Get(cluster, aeconditions.UpgradingCondition)
	clusterNotUpgrading := upgradingCondition != nil && upgradingCondition.Status == corev1.ConditionFalse
	clusterCreatedOrUpgraded :=
		upgradingCondition.Reason == CreationCompletedReason ||
			upgradingCondition.Reason == UpgradeCompletedReason

	// Let's try to get the current release from the last successful upgrade we
	// did.
	if clusterNotUpgrading && clusterCreatedOrUpgraded {
		// (2) Cluster is currently not in the Upgrading state.
		// (3) This cluster has already been created or upgraded successfully
		// to some release, let's check what was the latest version.
		message := DeserializeUpgradingConditionMessage(upgradingCondition.Message)
		latestReleaseVersion, err := semver.ParseTolerant(message.ReleaseVersion)
		if err != nil {
			return false, microerror.Mask(err)
		}
		desiredReleaseVersion, err := semver.ParseTolerant(key.ReleaseVersion(cluster))
		if err != nil {
			return false, microerror.Mask(err)
		}

		// Upgrade has been started! :)
		if desiredReleaseVersion.GT(latestReleaseVersion) {
			// (4) Desired release for this cluster is newer than the release
			// to which it was previously upgraded, which means that we have a
			// new upgrade to do.
			//
			// Based on (1), (2), (3) and (4), we can conclude that the cluster
			// should be in the Upgrading state
			return true, nil
		}
	}

	return false, nil
}

func setUpgradeNotStarted(cluster *capi.Cluster) {
	// Cluster is just being created, no upgrade yet.
	message := UpgradingConditionMessage{
		Message:        "Upgrade not started",
		ReleaseVersion: key.ReleaseVersion(cluster),
	}
	messageString := SerializeUpgradingConditionMessage(message)
	capiconditions.MarkFalse(
		cluster,
		aeconditions.UpgradingCondition,
		UpgradeNotStartedReason,
		capi.ConditionSeverityInfo,
		messageString)
}
