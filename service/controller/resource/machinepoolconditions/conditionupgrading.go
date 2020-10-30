package machinepoolconditions

import (
	"context"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/conditions"
	"github.com/giantswarm/azure-operator/v5/pkg/upgrade"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) ensureUpgradingCondition(ctx context.Context, machinePool *capiexp.MachinePool) error {
	r.logDebug(ctx, "ensuring condition Upgrading")
	var err error

	// Let's make sure that the condition status is set to a supported value.
	if conditions.IsUnexpected(machinePool, aeconditions.UpgradingCondition) {
		return conditions.UnexpectedConditionStatusError(machinePool, aeconditions.UpgradingCondition)
	}

	// Set initial Upgrading condition status to false, since the MachinePool
	// is just created.
	if capiconditions.IsUnknown(machinePool, aeconditions.UpgradingCondition) {
		err = conditions.MarkUpgradingNotStarted(machinePool)
	}

	// Let's now check if desired versions are set and deployed.
	desiredReleaseVersion := key.ReleaseVersion(machinePool)
	desiredAzureOperatorVersion := key.OperatorVersion(machinePool)

	upgradeIsCompletedForDesiredVersion, err := upgrade.IsNodePoolUpgradeCompleted(ctx, r.ctrlClient, machinePool, desiredReleaseVersion, desiredAzureOperatorVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	if conditions.IsUpgradingTrue(machinePool) && upgradeIsCompletedForDesiredVersion {
		// MachinePool was being upgraded, and the upgrade has been completed.
		err = conditions.MarkUpgradingCompleted(machinePool)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if conditions.IsUpgradingFalse(machinePool) && !upgradeIsCompletedForDesiredVersion {
		// Machine pool was not being upgraded, but upgrade is needed to reach
		// the desired version.
		conditions.MarkUpgradingStarted(machinePool)
	}

	r.logConditionStatus(ctx, machinePool, aeconditions.UpgradingCondition)
	r.logDebug(ctx, "ensured condition Upgrading")
	return nil
}
