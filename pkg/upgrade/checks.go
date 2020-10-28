package upgrade

import (
	"context"

	"github.com/coreos/go-semver/semver"
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsNodePoolUpgradeCompleted(ctx context.Context, c client.Client, machinePool *capiexp.MachinePool, desiredReleaseVersion, desiredAzureOperatorVersion string) (bool, error) {
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

	// Check MachinePool Upgrading condition
	isUpgrading := capiconditions.IsTrue(machinePool, aeconditions.UpgradingCondition)

	// Node pool is still being upgraded
	if isUpgrading {
		return false, nil
	}

	// And finally check the actual nodes
	anyNodePoolNodeOutOfDate, err := AnyNodePoolNodeOutOfDate(ctx, c, machinePool.Name, desiredAzureOperatorVersion)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// When all nodes are up-to-date, the upgrade has been completed
	upgradeCompleted := !anyNodePoolNodeOutOfDate

	return upgradeCompleted, nil
}

func AnyNodePoolNodeOutOfDate(ctx context.Context, c client.Client, nodepoolID string, desiredAzureOperatorVersion string) (bool, error) {
	nodes := &corev1.NodeList{}
	var labelSelector client.MatchingLabels
	{
		labelSelector = map[string]string{
			label.MachinePool: nodepoolID,
		}
	}

	err := c.List(ctx, nodes, labelSelector)
	if err != nil {
		return false, microerror.Mask(err)
	}

	desiredVersion := semver.New(desiredAzureOperatorVersion)

	for _, node := range nodes.Items {
		operatorVersionLabel, exists := node.GetLabels()[label.AzureOperatorVersion]
		if !exists {
			return true, nil
		}

		operatorVersion := semver.New(operatorVersionLabel)

		if operatorVersion.LessThan(*desiredVersion) {
			return true, nil
		}
	}

	return false, nil
}
