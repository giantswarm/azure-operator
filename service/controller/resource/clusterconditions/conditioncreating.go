package clusterconditions

import (
	"context"
	"fmt"
	"time"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
)

// ensureCreatingCondition ensures that the Cluster CR has Creation condition
// set. There are 3 cases for Creation condition status:
// (1) Unknown (or not yet set), when Cluster controller has not yet reconciled
// the Cluster CR, so the Creating condition is not yet set,
// (2) True, when Cluster creation is in progress, and
// (3) False, when Cluster creation has been completed.
func (r *Resource) ensureCreatingCondition(ctx context.Context, cluster *capi.Cluster) error {
	var err error

	// Creating condition is not set or it has Unknown status, let's set it for
	// the first time.
	if capiconditions.IsUnknown(cluster, aeconditions.CreatingCondition) {
		err = r.setCreatingCondition(ctx, cluster)
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

// setCreatingCondition sets Creating condition to its initial status True.
func (r *Resource) setCreatingCondition(ctx context.Context, cluster *capi.Cluster) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "Cluster creation started, setting Creating condition to True")
	capiconditions.MarkTrue(cluster, aeconditions.CreatingCondition)
	return nil
}

// updateCreatingCondition checks if the cluster creation has been completed,
// and if it is, it updates Creating condition status to False.
func (r *Resource) updateCreatingCondition(ctx context.Context, cluster *capi.Cluster) error {
	// We processed Unknown and False statuses for Creating condition, now we
	// expect that the only remaining option is True. If that is not the case,
	// then we have an unexpected status value for Creating condition.
	creatingCondition := capiconditions.Get(cluster, aeconditions.CreatingCondition)
	if creatingCondition.Status != corev1.ConditionTrue {
		return microerror.Maskf(invalidConditionError, "Expected that Cluster Creating conditions is True, got %s", creatingCondition.Status)
	}

	// Condition Creating is True, that means that the cluster creation is in
	// progress. Now we will check if the cluster creation has been completed.
	readyCondition := capiconditions.Get(cluster, capi.ReadyCondition)

	// Cluster is not Ready yet, creation is still in progress.
	if readyCondition.Status != corev1.ConditionTrue {
		r.logger.LogCtx(ctx, "level", "debug", "message", "Cluster not ready, condition Ready!=True, creation is still in progress")
		return nil
	}

	// Cluster is Ready, this should mean that the creation has been completed.
	r.logger.LogCtx(ctx, "level", "debug", "message", "Cluster condition Ready=True, creation should be completed")

	// Just a quick sanity check before we call it a win, cluster can become
	// ready only after the creation has been initiated.
	if readyCondition.LastTransitionTime.Before(&creatingCondition.LastTransitionTime) {
		errorMessageFormat := "Ready was set to True (at %s), before Creating was set to false (at %s), " +
			"that means that the cluster was ready before the creation has started, which is not possible"
		// log the error, in case the caller ignores it
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf(errorMessageFormat, readyCondition.LastTransitionTime, creatingCondition.LastTransitionTime))

		return microerror.Maskf(invalidConditionError, errorMessageFormat, readyCondition.LastTransitionTime, creatingCondition.LastTransitionTime)
	}

	// Cluster creation has been completed! :)

	// Let's see how long it took.
	clusterCreationDuration := time.Now().Sub(cluster.CreationTimestamp.Time)

	// Declaring this cluster officially created!
	capiconditions.MarkFalse(
		cluster,
		aeconditions.CreatingCondition,
		"CreationCompleted",
		capi.ConditionSeverityInfo,
		fmt.Sprintf("Cluster creation has been completed in %s", clusterCreationDuration))

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Cluster condition Creating set to False, creation has been completed after %s", clusterCreationDuration))
	return nil
}
