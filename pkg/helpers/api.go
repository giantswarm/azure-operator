package helpers

import (
	"context"

	apieconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
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
	// Note: This is alpha implementation that checks only VPN and node pool VMSS. Final
	// implementation should include checking of other Azure resources as well.
	var isAzureClusterReady bool
	var isVpnGatewayReadyCondition bool

	// Check if VPN gateway is ready
	isVpnGatewayReadyCondition = conditions.IsTrue(azureCluster, azureconditions.VPNGatewayReadyCondition)
	isAzureClusterReady = isVpnGatewayReadyCondition

	if isAzureClusterReady {
		conditions.MarkTrue(azureCluster, capiV1alpha3.ReadyCondition)
	} else {
		var conditionReason string
		var conditionMessage string

		if !isVpnGatewayReadyCondition {
			conditionReason = "VPNGatewayNotReady"
			conditionMessage = "VPN Gateway is not ready"
		} else {
			conditionReason = "UnknownReason"
			conditionMessage = "Cluster is not ready for an unexpected reason"
		}

		conditions.MarkFalse(
			azureCluster,
			capiV1alpha3.ReadyCondition,
			conditionReason,
			capiV1alpha3.ConditionSeverityWarning,
			conditionMessage)
	}

	err := c.Status().Update(ctx, azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	logger.LogCtx(ctx,
		"level", "debug",
		"type", "AzureCluster",
		"message", "set Status.Condition",
		"conditionType", capiV1alpha3.ReadyCondition,
		"conditionStatus", isAzureClusterReady)

	return nil
}

func UpdateClusterConditions(ctx context.Context, c client.Client, logger micrologger.Logger, cluster *capiV1alpha3.Cluster, azureCluster *capzV1alpha3.AzureCluster) error {
	// Note: This is alpha implementation. Final implementation should include checking of
	// MachinePool CRs (worker nodes).
	logger.LogCtx(ctx, "level", "debug", "type", "Cluster", "message", "setting Status.Condition", "conditionType", capiV1alpha3.ReadyCondition)

	if conditions.IsTrue(azureCluster, capiV1alpha3.ReadyCondition) {
		conditions.MarkTrue(cluster, capiV1alpha3.ReadyCondition)
	} else {
		conditions.MarkFalse(
			cluster,
			capiV1alpha3.ReadyCondition,
			"AzureClusterNotReady",
			capiV1alpha3.ConditionSeverityWarning,
			"AzureCluster is not yet ready")
	}

	creatingCompleted := false
	if conditions.IsTrue(cluster, apieconditions.CreatingCondition) && conditions.IsTrue(cluster, capiV1alpha3.ReadyCondition) {
		conditions.MarkFalse(
			cluster,
			apieconditions.CreatingCondition,
			"CreationCompleted",
			capiV1alpha3.ConditionSeverityInfo,
			"Cluster creation is completed")
		creatingCompleted = true
	}

	upgradingCompleted := false
	if conditions.IsTrue(cluster, apieconditions.UpgradingCondition) && conditions.IsTrue(cluster, capiV1alpha3.ReadyCondition) {
		// TODO: check Ready.LastUpdateTime > Upgrading.LastUpdateTime before setting Upgrading to False
		conditions.MarkFalse(
			cluster,
			apieconditions.UpgradingCondition,
			"UpgradeCompleted",
			capiV1alpha3.ConditionSeverityInfo,
			"Cluster upgrade is completed")
		upgradingCompleted = true
	}

	err := c.Status().Update(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	logger.LogCtx(ctx,
		"level", "debug",
		"type", "Cluster",
		"message", "set Status.Condition",
		"conditionType", capiV1alpha3.ReadyCondition,
		"conditionStatus", conditions.IsTrue(cluster, capiV1alpha3.ReadyCondition))

	if creatingCompleted {
		logger.LogCtx(ctx,
			"level", "debug",
			"type", "Cluster",
			"message", "set Status.Condition",
			"conditionType", apieconditions.CreatingCondition,
			"conditionStatus", metav1.ConditionTrue)
	}

	if upgradingCompleted {
		logger.LogCtx(ctx,
			"level", "debug",
			"type", "Cluster",
			"message", "set Status.Condition",
			"conditionType", apieconditions.UpgradingCondition,
			"conditionStatus", metav1.ConditionTrue)
	}

	return nil
}
