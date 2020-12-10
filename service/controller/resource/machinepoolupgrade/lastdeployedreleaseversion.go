package machinepoolupgrade

import (
	"context"
	"fmt"

	"github.com/coreos/go-semver/semver"
	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) ensureLastDeployedReleaseVersion(ctx context.Context, machinePool *capiexp.MachinePool) error {
	r.logger.Debugf(ctx, "ensuring release.giantswarm.io/last-deployed-version on MachinePool CR")
	desiredReleaseVersion := key.ReleaseVersion(machinePool)
	lastDeployedReleaseVersion := machinePool.Annotations[annotation.LastDeployedReleaseVersion]

	// release.giantswarm.io/last-deployed-version annotation is up-to-date.
	if lastDeployedReleaseVersion == desiredReleaseVersion {
		logMessage := fmt.Sprintf(
			"ensured release.giantswarm.io/last-deployed-version annotation, value %s it's already up-to-date",
			lastDeployedReleaseVersion)
		r.logger.Debugf(ctx, logMessage)
		return nil
	}

	desiredAzureOperatorVersion := key.OperatorVersion(machinePool)

	cluster, err := util.GetClusterFromMetadata(ctx, r.ctrlClient, machinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	tenantClusterClient, err := r.tenantClientFactory.GetClient(ctx, cluster)
	if tenantcluster.IsAPINotAvailableError(err) {
		r.logger.Debugf(ctx, "tenant API not available yet")
		r.logger.Debugf(ctx, "canceling resource")

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}
	upgradeIsCompletedForDesiredVersion, err := isNodePoolUpgradeCompleted(
		ctx,
		tenantClusterClient,
		machinePool,
		desiredReleaseVersion,
		desiredAzureOperatorVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	var logMessage string
	if upgradeIsCompletedForDesiredVersion {
		machinePool.Annotations[annotation.LastDeployedReleaseVersion] = key.ReleaseVersion(machinePool)
		err = r.ctrlClient.Update(ctx, machinePool)
		if err != nil {
			return microerror.Mask(err)
		}

		logMessage = fmt.Sprintf(
			"ensured release.giantswarm.io/last-deployed-version annotation, updated to %s",
			machinePool.Annotations[annotation.LastDeployedReleaseVersion])
	} else {
		logMessage = fmt.Sprintf(
			"ensured release.giantswarm.io/last-deployed-version annotation, upgrade is not completed, value is still %s",
			machinePool.Annotations[annotation.LastDeployedReleaseVersion])
	}

	r.logger.Debugf(ctx, logMessage)
	return nil
}

func isNodePoolUpgradeCompleted(ctx context.Context, tenantClusterClient client.Client, machinePool *capiexp.MachinePool, desiredReleaseVersion, desiredAzureOperatorVersion string) (bool, error) {
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
	allNodePoolNodesUpToDate, err := allNodePoolNodesUpToDate(ctx, tenantClusterClient, machinePool, desiredAzureOperatorVersion)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// When all nodes are up-to-date, the upgrade has been completed
	upgradeCompleted := allNodePoolNodesUpToDate

	return upgradeCompleted, nil
}

func allNodePoolNodesUpToDate(ctx context.Context, tenantClusterClient client.Client, machinePool *capiexp.MachinePool, desiredAzureOperatorVersion string) (bool, error) {
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
