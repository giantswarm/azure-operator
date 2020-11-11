package clusterreleaseversion

import (
	"context"
	"time"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/conditions"
)

func (r *Resource) isUpgradeCompleted(ctx context.Context, cluster *capi.Cluster) (bool, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "checking if cluster upgrade has been completed")
	// Here we expect that Cluster Upgrading conditions is set to True.
	upgradingCondition := capiconditions.Get(cluster, aeconditions.UpgradingCondition)
	if !conditions.IsTrue(upgradingCondition) {
		err := microerror.Maskf(
			conditions.UnexpectedConditionStatusError,
			conditions.ExpectedTrueErrorMessage(cluster, aeconditions.UpgradingCondition))
		return false, err
	}

	// Upgrading is in progress, now let's check if it has been completed.

	// But don't check if Upgrading has been completed for the first 5 minutes,
	// give other controllers time to start reconciling their CRs.
	if time.Now().Before(upgradingCondition.LastTransitionTime.Add(5 * time.Minute)) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "upgrade is in progress for less than 5 minutes, check back later")
		return false, nil
	}

	// Cluster has been in Upgrading state for at least 5 minutes now, so
	// let's check if it is Ready.
	readyCondition := capiconditions.Get(cluster, capi.ReadyCondition)

	if !conditions.IsTrue(readyCondition) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "cluster not ready, upgrade is still in progress")
		return false, nil
	}

	// (1) In addition to cluster being ready, here we check that it actually
	// became ready during the upgrade, which would mean that the upgrade has
	// been completed.
	becameReadyWhileUpgrading := readyCondition.LastTransitionTime.After(upgradingCondition.LastTransitionTime.Time)

	// (2) Or we declare Upgrading to be completed if nothing happened for 20
	// minutes, which could currently happen if we were upgrading some
	// component which is not covered by any Ready status condition.
	const upgradingWithoutReadyUpdateThreshold = 20 * time.Minute
	isReadyDuringEntireUpgradeProcess := time.Now().After(upgradingCondition.LastTransitionTime.Add(upgradingWithoutReadyUpdateThreshold))

	if becameReadyWhileUpgrading || isReadyDuringEntireUpgradeProcess {
		// Cluster is ready, and either (1) or (2) is true, so we consider the upgrade to be completed
		r.logger.LogCtx(ctx, "level", "debug", "message", "cluster upgrade has been completed")
		return true, nil
	}

	// Cluster is Ready, but since neither (1) nor (2) were satisfied, we wait
	// more before considering the upgrade to be completed
	r.logger.LogCtx(ctx, "level", "debug", "message", "cluster is ready, but it's too soon to tell if the upgrade has been completed")
	return false, nil
}
