package clusterreleaseversion

import (
	"context"
	"fmt"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/conditions"
)

// isCreationCompleted checks if the cluster creation has been completed.
func (r *Resource) isCreationCompleted(ctx context.Context, cluster *capi.Cluster) (bool, error) {
	r.logger.Debugf(ctx, "checking if cluster creation has been completed")

	// Here we expect that Cluster Creating conditions is set to True.
	creatingCondition := capiconditions.Get(cluster, aeconditions.CreatingCondition)
	if !conditions.IsTrue(creatingCondition) {
		err := microerror.Maskf(
			conditions.UnexpectedConditionStatusError,
			conditions.ExpectedTrueErrorMessage(cluster, aeconditions.CreatingCondition))
		return false, err
	}

	// Now let's check if the cluster became Ready, which should mean that the
	// creation was completed.
	readyCondition := capiconditions.Get(cluster, capi.ReadyCondition)

	if !conditions.IsTrue(readyCondition) {
		r.logger.Debugf(ctx, "Cluster not ready, condition Ready!=True, creation is still in progress")
		return false, nil
	}

	// Cluster is Ready, this should mean that the creation has been completed.
	r.logger.Debugf(ctx, "Cluster condition Ready=True, creation should be completed")

	// Just a quick sanity check before we call it a win, cluster can become
	// ready only after the creation has been initiated.
	if readyCondition.LastTransitionTime.Before(&creatingCondition.LastTransitionTime) {
		errorMessageFormat := "Ready was set to True (at %s), before Creating was set to false (at %s), " +
			"that means that the cluster was ready before the creation has started, which is not possible"
		// log the error, in case the caller ignores it
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf(errorMessageFormat, readyCondition.LastTransitionTime, creatingCondition.LastTransitionTime))
		err := microerror.Maskf(
			conditions.UnexpectedConditionStatusError,
			errorMessageFormat,
			readyCondition.LastTransitionTime,
			creatingCondition.LastTransitionTime)
		return false, err
	}

	// Cluster creation has been completed! :)
	r.logger.Debugf(ctx, "cluster creation has been completed")
	return true, nil
}
