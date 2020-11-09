package machinepoolconditions

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/conditions"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) ensureUpgradingCondition(ctx context.Context, machinePool *capiexp.MachinePool) error {
	r.logDebug(ctx, "ensuring condition Upgrading")

	// Let's make sure that the condition status is set to a supported value.
	if conditions.IsUnexpected(machinePool, aeconditions.UpgradingCondition) {
		return microerror.Maskf(
			conditions.UnexpectedConditionStatusError,
			conditions.UnexpectedConditionStatusErrorMessage(machinePool, aeconditions.UpgradingCondition))
	}

	lastDeployedReleaseVersion, isLastDeployedReleaseVersionSet := machinePool.Annotations[annotation.LastDeployedReleaseVersion]
	if !isLastDeployedReleaseVersionSet {
		// Node pool is being created.
		conditions.MarkUpgradingNotStarted(machinePool)
		r.logConditionStatus(ctx, machinePool, aeconditions.UpgradingCondition)
		r.logDebug(ctx, "ensured condition Upgrading, cluster is being created")
		return nil
	}

	// Let's now check if desired release version is deployed.
	desiredReleaseVersion := key.ReleaseVersion(machinePool)
	desiredReleaseVersionIsDeployed := lastDeployedReleaseVersion == desiredReleaseVersion

	if capiconditions.IsUnknown(machinePool, aeconditions.UpgradingCondition) {
		// MachinePool CR is still being created, or it's restored from backup,
		// this case should be very rare and almost never happen.
		if desiredReleaseVersionIsDeployed {
			conditions.MarkUpgradingNotStarted(machinePool)
		} else {
			conditions.MarkUpgradingStarted(machinePool)
		}
	} else if conditions.IsUpgradingTrue(machinePool) && desiredReleaseVersionIsDeployed {
		// MachinePool was being upgraded, and the upgrade has been completed.
		conditions.MarkUpgradingCompleted(machinePool)
	} else if conditions.IsUpgradingFalse(machinePool) && !desiredReleaseVersionIsDeployed {
		// Machine pool was not being upgraded, but upgrade is needed to reach
		// the desired version.
		conditions.MarkUpgradingStarted(machinePool)
	}

	r.logConditionStatus(ctx, machinePool, aeconditions.UpgradingCondition)
	r.logDebug(ctx, "ensured condition Upgrading")
	return nil
}
