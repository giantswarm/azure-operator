package clusterconditions

import (
	"context"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
)

func (r *Resource) ensureProviderInfrastructureReadyCondition(ctx context.Context, cluster *capi.Cluster) error {
	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, r.ctrlClient, cluster.ObjectMeta)
	if apierrors.IsNotFound(err) {
		warningMessage := "AzureCluster CR %s in namespace %s is not found"
		warningMessageArgs := []interface{}{cluster.Name, cluster.Namespace}
		r.logWarning(ctx, warningMessage, warningMessageArgs...)

		capiconditions.MarkFalse(
			cluster,
			capi.ReadyCondition,
			"AzureClusterNotFound",
			capi.ConditionSeverityWarning,
			warningMessage,
			warningMessageArgs...)

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	fallbackToFalse := capiconditions.WithFallbackValue(
		false,
		"AzureClusterConditionReadyNotSet",
		capi.ConditionSeverityWarning,
		"AzureCluster Ready condition is not yet set, check again in few minutes")
	capiconditions.SetMirror(cluster, aeconditions.ProviderInfrastructureReadyCondition, azureCluster, fallbackToFalse)

	cluster.Status.InfrastructureReady = capiconditions.IsTrue(azureCluster, capi.ReadyCondition)

	return nil
}
