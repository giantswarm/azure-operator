package upgrade

import (
	"context"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	azopannotation "github.com/giantswarm/azure-operator/v5/pkg/annotation"
	"github.com/giantswarm/azure-operator/v5/pkg/conditions"
)

// IsFirstNodePoolUpgradeInProgress checks if the cluster is being upgraded
// from an old/legacy release to the node pools release.
func IsFirstNodePoolUpgradeInProgress(cluster *capi.Cluster) bool {
	upgradingToNodePools, isUpgradingToNodePoolsSet := cluster.GetAnnotations()[azopannotation.UpgradingToNodePools]
	return isUpgradingToNodePoolsSet && strings.ToLower(upgradingToNodePools) == "true"
}

func IsNodePoolUpgradeInProgressOrPending(ctx context.Context, c client.Client, machinePool *capiexp.MachinePool, desiredReleaseVersion, desiredAzureOperatorVersion string) (bool, error) {
	if conditions.IsUpgradingTrue(machinePool) {
		// Upgrade is in progress.
		return true, nil
	}

	isNodePoolUpgradeCompleted, err := IsNodePoolUpgradeCompleted(ctx, c, machinePool, desiredReleaseVersion, desiredAzureOperatorVersion)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// If the upgrade has not been completed for the desired release, then we
	// have a pending upgrade to do.
	upgradeIsPending := !isNodePoolUpgradeCompleted

	return upgradeIsPending, nil
}

func IsNodePoolUpgradeCompleted(ctx context.Context, tenantClusterClient client.Client, machinePool *capiexp.MachinePool, desiredReleaseVersion, desiredAzureOperatorVersion string) (bool, error) {
	// Check desired release version
	currentReleaseVersion := machinePool.GetLabels()[label.ReleaseVersion]
	if currentReleaseVersion != desiredReleaseVersion {
		return false, nil
	}

	// Check desired azure-operator version
	currentAzureOperatorVersion := machinePool.GetLabels()[label.AzureOperatorVersion]
	if currentAzureOperatorVersion != desiredAzureOperatorVersion {
		return false, nil
	}

	// And finally check the actual nodes
	allNodePoolNodesUpToDate, err := AllNodePoolNodesUpToDate(ctx, tenantClusterClient, machinePool, desiredAzureOperatorVersion)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// When all nodes are up-to-date, the upgrade has been completed
	upgradeCompleted := allNodePoolNodesUpToDate

	return upgradeCompleted, nil
}

func AllNodePoolNodesUpToDate(ctx context.Context, tenantClusterClient client.Client, machinePool *capiexp.MachinePool, desiredAzureOperatorVersion string) (bool, error) {
	nodes := &corev1.NodeList{}
	err := tenantClusterClient.List(ctx, nodes, client.MatchingLabels{label.MachinePool: machinePool.Name})
	if err != nil {
		return false, microerror.Mask(err)
	}

	desiredVersion := semver.New(desiredAzureOperatorVersion)
	var upToDateNodesCount int32
	var outdatedNodes int32

	for _, node := range nodes.Items {
		nodeOperatorVersionLabel, exists := node.GetLabels()[label.AzureOperatorVersion]
		if !exists {
			return false, nil
		}

		nodeOperatorVersion := semver.New(nodeOperatorVersionLabel)

		if nodeOperatorVersion.LessThan(*desiredVersion) {
			outdatedNodes++
		} else {
			upToDateNodesCount++
		}
	}

	// azure-admission-controller ensures that machinePool.Spec.Replicas is
	// always set
	requiredReplicas := *machinePool.Spec.Replicas

	// We want that all required replicas are up-to-date.
	requiredReplicasAreUpToDate := upToDateNodesCount >= requiredReplicas

	// We also want that old nodes are removed.
	oldNodesAreRemoved := outdatedNodes == 0

	return requiredReplicasAreUpToDate && oldNodesAreRemoved, nil
}
