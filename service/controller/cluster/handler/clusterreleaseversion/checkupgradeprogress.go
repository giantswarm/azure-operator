package clusterreleaseversion

import (
	"context"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/conditions/pkg/conditions"
	"github.com/giantswarm/microerror"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

	controlPlaneUpgraded, err := r.isControlPlaneUpgraded(ctx, cluster)
	if err != nil {
		return false, microerror.Mask(err)
	}
	if !controlPlaneUpgraded {
		r.logger.Debugf(ctx, "cluster upgrade is still in progress")
		return false, nil
	}

	allNodePoolsUpgraded, err := r.areNodePoolsUpgraded(ctx, cluster)
	if err != nil {
		return false, microerror.Mask(err)
	}
	if !allNodePoolsUpgraded {
		r.logger.Debugf(ctx, "cluster upgrade is still in progress")
		return false, nil
	}

	r.logger.Debugf(ctx, "cluster upgrade has been completed")
	return true, nil
}

func (r *Resource) isControlPlaneUpgraded(ctx context.Context, cluster *capi.Cluster) (bool, error) {
	r.logger.Debugf(ctx, "checking if control plane upgrade has been completed")

	azureMachineList := capz.AzureMachineList{}
	err := r.ctrlClient.List(ctx, &azureMachineList, client.MatchingLabels{capi.ClusterLabelName: cluster.Name})
	if err != nil {
		return false, microerror.Mask(err)
	}

	// This is the desired cluster release version. We will check if control
	// plane nodes are upgraded to this release version.
	desiredClusterReleaseVersion := key.ReleaseVersion(cluster)

	for i := range azureMachineList.Items {
		if !isUpgradedToDesiredReleaseVersion(&azureMachineList.Items[i], desiredClusterReleaseVersion) {
			r.logger.Debugf(ctx, "AzureMachine %s has not been upgraded yet", azureMachines.Name)
			return false, nil
		}
	}

	r.logger.Debugf(ctx, "control plane upgrade has been completed")
	return true, nil
}

func (r *Resource) areNodePoolsUpgraded(ctx context.Context, cluster *capi.Cluster) (bool, error) {
	r.logger.Debugf(ctx, "checking if node pools upgrade has been completed")
	machinePools, err := helpers.GetMachinePoolsByMetadata(ctx, r.ctrlClient, cluster.ObjectMeta)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// This is the desired cluster release version. We will check if node pools
	// are upgraded to this release version.
	desiredClusterReleaseVersion := key.ReleaseVersion(cluster)

	for _, machinePool := range machinePools.Items {
		if !isUpgradedToDesiredReleaseVersion(&machinePool, desiredClusterReleaseVersion) {
			r.logger.Debugf(ctx, "MachinePool %s has not been upgraded yet", machinePool.Name)
			return false, nil
		}
	}

	r.logger.Debugf(ctx, "node pools upgrade has been completed")
	return true, nil
}

func isUpgradedToDesiredReleaseVersion(obj conditions.Object, desiredClusterReleaseVersion string) bool {
	if conditions.IsCreatingTrue(obj) {
		// Keeping this for backward compatibility, but it should not be necessary:
		// We are in the upgrade process, but a node pool is being created. This
		// is the case for first upgrade to node pools release, so an upgrade
		// from v12.x.x to v13.x.x, as cluster upgrade will trigger first node
		// pool creation.
		return false
	}

	if conditions.IsUpgradingTrue(obj) {
		// Control plane node or node pool is being upgraded.
		return false
	}

	currentDesiredReleaseVersion := key.ReleaseVersion(obj)
	if currentDesiredReleaseVersion != desiredClusterReleaseVersion {
		// A control plane node or node pool upgrade has not been started yet.
		return false
	}

	lastDeployedReleaseVersion, ok := obj.GetAnnotations()[annotation.LastDeployedReleaseVersion]
	if !ok {
		// Control plane or a node pool is still not created. This should be
		// caught above in Creating check, but let's err on the side of caution
		// here.
		return false
	}

	if lastDeployedReleaseVersion != desiredClusterReleaseVersion {
		// A control plane node or node pool has not yet been upgraded to the
		// desired release version.
		return false
	}

	// Upgrade completed for control plane node or node pool!
	return true
}
