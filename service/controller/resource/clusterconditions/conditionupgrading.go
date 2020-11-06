package clusterconditions

import (
	"context"

	"github.com/blang/semver"
	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/conditions"
	"github.com/giantswarm/azure-operator/v5/pkg/upgrade"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) ensureUpgradingCondition(ctx context.Context, cluster *capi.Cluster) error {
	if conditions.IsUnexpected(cluster, aeconditions.UpgradingCondition) {
		return microerror.Maskf(
			conditions.UnexpectedConditionStatusError,
			conditions.UnexpectedConditionStatusErrorMessage(cluster, aeconditions.UpgradingCondition))
	}

	// Case 1: new cluster just being created, no upgrade yet.
	if capiconditions.IsTrue(cluster, aeconditions.CreatingCondition) {
		conditions.MarkUpgradingNotStarted(cluster)
		return nil
	}

	// Case 2: first upgrade to node pools release, a bit of an edgecase,
	// albeit an important one :)
	if upgrade.IsFirstNodePoolUpgradeInProgress(cluster) {
		if !capiconditions.IsTrue(cluster, aeconditions.UpgradingCondition) {
			conditions.MarkUpgradingStarted(cluster)
		}
		return nil
	}

	// Let's check what was the last release version that we successfully deployed.
	lastDeployedReleaseVersionString, isLastDeployedReleaseVersionSet := cluster.GetAnnotations()[annotation.LastDeployedReleaseVersion]
	if !isLastDeployedReleaseVersionSet {
		// Last deployed release version annotation is not set at all, which
		// means that cluster creation has not completed, so no upgrades yet.
		// This case should be already processed by Creating condition handler,
		// and we would have Case 1 from above, but here we check just in case.
		conditions.MarkUpgradingNotStarted(cluster)
		return nil
	}

	// Last deployed release version annotation is set, which means that this
	// cluster has already been created or upgraded successfully to some
	// release, let's check what we have and compare it to what we want.

	latestDeployedReleaseVersion, err := semver.ParseTolerant(lastDeployedReleaseVersionString)
	if err != nil {
		return microerror.Mask(err)
	}

	desiredReleaseVersion, err := semver.ParseTolerant(key.ReleaseVersion(cluster))
	if err != nil {
		return microerror.Mask(err)
	}

	if capiconditions.IsTrue(cluster, aeconditions.UpgradingCondition) {
		// Case 3: Upgrading=True, upgrade is currently in progress, here
		// we check if it has been completed.

		if latestDeployedReleaseVersion.EQ(desiredReleaseVersion) {
			// Last deployed release version for this cluster is equal to the
			// desired release version, so we can conclude that the cluster
			// upgrade has been completed.
			conditions.MarkUpgradingCompleted(cluster)
		}
	} else {
		// Case 4: Upgrading is Unknown or False, let's check if the cluster is
		// being upgraded.
		if desiredReleaseVersion.GT(latestDeployedReleaseVersion) {
			// Desired release for this cluster is newer than the release to
			// which it was previously upgraded or with which was created, so
			// we can conclude that the cluster is in the Upgrading state.
			conditions.MarkUpgradingStarted(cluster)
		} else {
			// Desired release for this cluster is equal to the release to
			// which it was previously upgraded or with which was created, so
			// we can conclude that the cluster upgrade has not started.
			//
			// Note: desired release version cannot be less than last deployed
			// release version, since we don't allow release version downgrades,
			// which is validated by azure-admission-controller.
			conditions.MarkUpgradingNotStarted(cluster)
		}
	}

	return nil
}
