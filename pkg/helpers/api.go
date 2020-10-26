package helpers

import (
	"context"

	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capzV1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiV1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
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
