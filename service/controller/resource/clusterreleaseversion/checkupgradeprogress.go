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

	if !conditions.IsTrue(readyCondition) {
		r.logger.Debugf(ctx, "cluster not ready, upgrade is still in progress")
		return false, nil
	}

	// (1) In addition to cluster being ready, here we check that it actually
	// became ready during the upgrade, which would mean that the upgrade has
	// been completed.
	becameReadyWhileUpgrading := readyCondition.LastTransitionTime.After(upgradingCondition.LastTransitionTime.Time)

	machinePools, err := helpers.GetMachinePoolsByMetadata(ctx, r.ctrlClient, cluster.ObjectMeta)
	if err != nil {
		return false, microerror.Mask(err)
	}

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

	// (2) Or we declare Upgrading to be completed if nothing happened for 20
	// minutes, which could currently happen if we were upgrading some
	// component which is not covered by any Ready status condition.
	const upgradingWithoutReadyUpdateThreshold = 45 * time.Minute
	isReadyDuringEntireUpgradeProcess := time.Now().After(upgradingCondition.LastTransitionTime.Add(upgradingWithoutReadyUpdateThreshold))

	if (becameReadyWhileUpgrading && allNodePoolsUpgraded) || isReadyDuringEntireUpgradeProcess {
		// Cluster is ready, and either (1) or (2) is true, so we consider the upgrade to be completed
		r.logger.Debugf(ctx, "cluster upgrade has been completed")
		return true, nil
	}

	// Cluster is Ready, but since neither (1) nor (2) were satisfied, we wait
	// more before considering the upgrade to be completed
	r.logger.Debugf(ctx, "cluster is ready, but it's too soon to tell if the upgrade has been completed")
	return false, nil
}
