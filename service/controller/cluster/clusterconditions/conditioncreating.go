package clusterconditions

import (
	"context"
	"fmt"
	"time"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/conditions"
	"github.com/giantswarm/azure-operator/v5/pkg/upgrade"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// ensureCreatingCondition ensures that the Cluster CR has Creation condition
// set. There are 3 cases for Creation condition status:
// (1) Unknown (or not yet set), when Cluster controller has not yet reconciled
// the Cluster CR, so the Creating condition is not yet set,
// (2) True, when Cluster creation is in progress, and
// (3) False, when Cluster creation has been completed.
func (r *Resource) ensureCreatingCondition(ctx context.Context, cluster *capi.Cluster) error {
	if conditions.IsUnexpected(cluster, aeconditions.CreatingCondition) {
		return microerror.Maskf(
			conditions.UnexpectedConditionStatusError,
			conditions.UnexpectedConditionStatusErrorMessage(cluster, aeconditions.CreatingCondition))
	}

	var err error

	// Creating condition is not set or it has Unknown status, let's set it for
	// the first time.
	if capiconditions.IsUnknown(cluster, aeconditions.CreatingCondition) {
		err = r.initializeCreatingCondition(ctx, cluster)
		if err != nil {
			return microerror.Mask(err)
		}
		return nil
	}

	// Creating condition is False, which means that the cluster creation is
	// completed, so we don't have to update it anymore.
	if capiconditions.IsFalse(cluster, aeconditions.CreatingCondition) {
		// no logging for already created clusters, let's not spam unnecessarily
		return nil
	}

	// Cluster creation should be completed, let's check that and update
	// Creating condition.
	err = r.updateCreatingCondition(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// initializeCreatingCondition sets Creating condition to its initial status,
// which should happen in 3 cases, (1) when the cluster is just created, (2)
// when the pre-nodepools cluster is upgraded to a nodepools cluster, and (3)
// when a Cluster CR is restored from a backup.
func (r *Resource) initializeCreatingCondition(ctx context.Context, cluster *capi.Cluster) error {
	lastDeployedReleaseVersion, isLastDeployedReleaseVersionSet := cluster.GetAnnotations()[annotation.LastDeployedReleaseVersion]

	if isLastDeployedReleaseVersionSet || upgrade.IsFirstNodePoolUpgradeInProgress(cluster) {
		message := "Cluster was already created or upgraded"
		if isLastDeployedReleaseVersionSet {
			// release.giantswarm.io/last-deployed-version annotation is set, which
			// means that the cluster is already created
			message += fmt.Sprintf("with release version %s", lastDeployedReleaseVersion)
		}

		capiconditions.MarkFalse(
			cluster,
			aeconditions.CreatingCondition,
			aeconditions.ExistingClusterReason,
			capi.ConditionSeverityInfo,
			message)

		logMessage := fmt.Sprintf("%s, setting Creating condition to False", message)
		r.logger.LogCtx(ctx, "level", "debug", "message", logMessage)
	} else {
		// release.giantswarm.io/last-deployed-version annotation is not set,
		// which means that the cluster is just being created.
		r.logger.LogCtx(ctx, "level", "debug", "message", "Cluster is just being created, setting Creating condition to True")
		capiconditions.MarkTrue(cluster, aeconditions.CreatingCondition)
	}

	return nil
}

// updateCreatingCondition checks if the cluster creation has been completed,
// and if it is, it updates Creating condition status to False.
func (r *Resource) updateCreatingCondition(ctx context.Context, cluster *capi.Cluster) error {
	// We processed Unknown and False statuses for Creating condition, now we
	// expect that the only remaining option is True. If that is not the case,
	// then we have an unexpected status value for Creating condition.
	if !capiconditions.IsTrue(cluster, aeconditions.CreatingCondition) {
		return microerror.Maskf(
			conditions.UnexpectedConditionStatusError,
			conditions.ExpectedTrueErrorMessage(cluster, aeconditions.CreatingCondition))
	}

	lastDeployedReleaseVersion, isSet := cluster.GetAnnotations()[annotation.LastDeployedReleaseVersion]
	if !isSet {
		// Cluster creation is not completed, since there is no last deployed
		// release version set.
		return nil
	}

	desiredReleaseVersion := key.ReleaseVersion(cluster)
	if lastDeployedReleaseVersion == desiredReleaseVersion {
		// Cluster creation has been completed! :)

		// Let's see how long it took.
		clusterCreationDuration := time.Since(cluster.CreationTimestamp.Time)

		// Declaring this cluster officially created!
		capiconditions.MarkFalse(
			cluster,
			aeconditions.CreatingCondition,
			aeconditions.CreationCompletedReason,
			capi.ConditionSeverityInfo,
			fmt.Sprintf("Cluster creation has been completed in %s", clusterCreationDuration))

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Cluster condition Creating set to False, creation has been completed after %s", clusterCreationDuration))
	}

	return nil
}
