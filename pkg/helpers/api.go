package helpers

import (
	"context"

	azureconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzV1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiV1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetAzureClusterFromMetadata returns the AzureCluster object (if present) using the object metadata.
func GetAzureClusterFromMetadata(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*capzV1alpha3.AzureCluster, error) {
	// Check if "cluster.x-k8s.io/cluster-name" label is set.
	if obj.Labels[capiV1alpha3.ClusterLabelName] == "" {
		err := microerror.Maskf(invalidObjectError, "Label %q must not be empty for object %q", capiV1alpha3.ClusterLabelName, obj.GetSelfLink())
		return nil, microerror.Mask(err)
	}

	return GetAzureClusterByName(ctx, c, obj.Namespace, obj.Labels[capiV1alpha3.ClusterLabelName])
}

// GetAzureClusterByName finds and return a AzureCluster object using the specified params.
func GetAzureClusterByName(ctx context.Context, c client.Client, namespace, name string) (*capzV1alpha3.AzureCluster, error) {
	azureCluster := &capzV1alpha3.AzureCluster{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.Get(ctx, key, azureCluster); err != nil {
		return nil, microerror.Mask(err)
	}

	return azureCluster, nil
}

func GetAzureMachinePoolsByMetadata(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*capzexp.AzureMachinePoolList, error) {
	if obj.Labels[capiV1alpha3.ClusterLabelName] == "" {
		err := microerror.Maskf(invalidObjectError, "Label %q must not be empty for object %q", capiV1alpha3.ClusterLabelName, obj.GetSelfLink())
		return nil, microerror.Mask(err)
	}

	azureMachinePools, err := GetAzureMachinePoolsByClusterID(ctx, c, obj.Namespace, obj.Labels[capiV1alpha3.ClusterLabelName])
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureMachinePools, nil
}

func GetAzureMachinePoolsByClusterID(ctx context.Context, c client.Client, clusterNamespace, clusterID string) (*capzexp.AzureMachinePoolList, error) {
	azureMachinePools := &capzexp.AzureMachinePoolList{}
	var labelSelector client.MatchingLabels
	{
		labelSelector = map[string]string{
			capiV1alpha3.ClusterLabelName: clusterID,
		}
	}

	err := c.List(ctx, azureMachinePools, labelSelector, client.InNamespace(clusterNamespace))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureMachinePools, nil
}

func UpdateAzureClusterConditions(ctx context.Context, c client.Client, logger micrologger.Logger, azureCluster *capzV1alpha3.AzureCluster) error {
	logger.LogCtx(ctx, "level", "debug", "type", "AzureCluster", "message", "setting Status.Condition", "conditionType", capiV1alpha3.ReadyCondition)
	// Note: This is an incomplete implementation that checks only resource
	// group, because it's created in the beginning, and the VPN Gateway,
	// because it's created at the end. Final implementation should include
	// checking of other Azure resources as well. and it will be done in
	// AzureCluster controller.

	// List of conditions that all need to be True for the Ready condition to be True
	conditionsToSummarize := conditions.WithConditions(
		azureconditions.ResourceGroupReadyCondition,
		azureconditions.VPNGatewayReadyCondition)

	conditions.SetSummary(
		azureCluster,
		conditionsToSummarize,
		conditions.AddSourceRef())

	err := c.Status().Update(ctx, azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	readyCondition := conditions.Get(azureCluster, capiV1alpha3.ReadyCondition)

	if readyCondition != nil {
		logger.LogCtx(ctx,
			"level", "debug",
			"type", "AzureCluster",
			"message", "set Status.Condition",
			"conditionType", capiV1alpha3.ReadyCondition,
			"conditionStatus", readyCondition.Status)
	} else {
		logger.LogCtx(ctx,
			"level", "debug",
			"type", "AzureCluster",
			"message", "Ready condition not set",
			"conditionType", capiV1alpha3.ReadyCondition)
	}

	return nil
}
