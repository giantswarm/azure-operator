package clusterreleaseversion

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/conditions/pkg/conditions"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) isUpgradeCompleted(ctx context.Context, cluster *capi.Cluster) (bool, error) {
	r.logger.Debugf(ctx, "checking if cluster upgrade has been completed")
	// Here we expect that Cluster Upgrading conditions is set to True.
	upgradingCondition, upgradingConditionSet := conditions.GetUpgrading(cluster)
	if !upgradingConditionSet || !conditions.IsTrue(&upgradingCondition) {
		err := microerror.Maskf(
			conditions.UnexpectedConditionStatusError,
			conditions.ExpectedTrueErrorMessage(cluster, conditions.Upgrading))
		return false, err
	}

	// Upgrading is in progress, now let's check if it has been completed.

	// But don't check if Upgrading has been completed for the first 5 minutes,
	// give other controllers time to start reconciling their CRs.
	if time.Now().Before(upgradingCondition.LastTransitionTime.Add(5 * time.Minute)) {
		r.logger.Debugf(ctx, "upgrade is in progress for less than 5 minutes, check back later")
		return false, nil
	}

	// Cluster has been in Upgrading state for at least 5 minutes now, so
	// let's check if it is Ready.
	readyCondition := capiconditions.Get(cluster, capi.ReadyCondition)
	isClusterReady := conditions.IsTrue(readyCondition)

	// In addition to cluster being ready, here we check that it actually became
	// ready during the upgrade, which would mean that the changes that were
	// happening due to the upgrade has been completed.
	becameReadyWhileUpgrading := isClusterReady && readyCondition.LastTransitionTime.After(upgradingCondition.LastTransitionTime.Time)

	// Now let's check if machine pools are upgraded. Since the control plane
	// is upgraded before the machine pools, when all machine pools are upgraded
	// that means that all cluster descendants (CP and node pools) are upgraded.
	machinePools, err := helpers.GetMachinePoolsByMetadata(ctx, r.ctrlClient, cluster.ObjectMeta)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// This is the desired cluster release version. We will check if node pools
	// are upgraded to this release version.
	desiredClusterReleaseVersion := key.ReleaseVersion(cluster)

	allNodePoolsUpgraded := true
	for _, machinePool := range machinePools.Items {
		if conditions.IsCreatingTrue(&machinePool) {
			// A node pool is being created, this is the case for first upgrade
			// to node pools release, as cluster upgrade will trigger first node
			// pool creation.
			allNodePoolsUpgraded = false
			break
		}

		if conditions.IsUpgradingTrue(&machinePool) {
			// A node pool is being upgraded.
			allNodePoolsUpgraded = false
			break
		}

		desiredMachinePoolReleaseVersion := key.ReleaseVersion(&machinePool)
		if desiredMachinePoolReleaseVersion != desiredClusterReleaseVersion {
			// A node pool upgrade has not been started yet.
			allNodePoolsUpgraded = false
			break
		}

		machinePoolLastDeployedReleaseVersion, machinePoolLastDeployedReleaseVersionSet := machinePool.Annotations[annotation.LastDeployedReleaseVersion]
		if !machinePoolLastDeployedReleaseVersionSet {
			// A node pool is still not created. This should be caught above in
			// Creating check, but let's err on the side of caution here.
			allNodePoolsUpgraded = false
			break
		}

		if machinePoolLastDeployedReleaseVersion != desiredClusterReleaseVersion {
			// A node pool has not yet been upgraded to the desired release version.
			allNodePoolsUpgraded = false
			break
		}
	}

	// (1) Cluster became ready during the upgrade and the upgrade has been
	// completed for all cluster descendants.
	// This is basically upgrade happy path.
	clusterIsReadyAndUpgraded := becameReadyWhileUpgrading && allNodePoolsUpgraded
	if clusterIsReadyAndUpgraded {
		r.logger.Debugf(ctx, "cluster became ready during the upgrade, all cluster descendants are upgraded, upgrade completed")
		return true, nil
	}

	// (2) Or we declare Upgrading to be completed if:
	// - cluster is ready,
	// - all cluster descendants are upgraded and
	// - nothing happened for 15 minutes,
	// which could currently happen if we were upgrading some component which
	// is probably not covered by Ready nor Upgrading condition.
	// This is currently the case when we are upgrading an app and there are no
	// changes where we need to roll the nodes.
	const upgradingTimeoutWhenReadyAndUpgraded = 15 * time.Minute
	readyAndUpgradedTimeoutReached :=
		isClusterReady &&
			allNodePoolsUpgraded &&
			time.Now().After(upgradingCondition.LastTransitionTime.Add(upgradingTimeoutWhenReadyAndUpgraded))
	if readyAndUpgradedTimeoutReached {
		r.logger.Debugf(ctx, "cluster is ready, all cluster descendants are upgraded, upgrade completed")
		return true, nil
	}

	// Cluster upgrade is still in progress.
	r.logger.Debugf(ctx, "cluster upgrade is still in progress")
	return false, nil
}
