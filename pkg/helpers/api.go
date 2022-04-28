package helpers

import (
	"context"

	oldcapiexp "github.com/giantswarm/apiextensions/v6/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetMachinePoolsByMetadata(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*capiexp.MachinePoolList, error) {
	if obj.Labels[capi.ClusterLabelName] == "" {
		err := microerror.Maskf(invalidObjectError, "Label %q must not be empty for object %q", capi.ClusterLabelName, obj.GetSelfLink())
		return nil, err
	}

	machinePools, err := GetMachinePoolsByClusterID(ctx, c, obj.Namespace, obj.Labels[capi.ClusterLabelName])
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return machinePools, nil
}

func GetMachinePoolsByClusterID(ctx context.Context, c client.Client, clusterNamespace, clusterID string) (*capiexp.MachinePoolList, error) {
	machinePools := &capiexp.MachinePoolList{}
	var labelSelector client.MatchingLabels
	{
		labelSelector = map[string]string{
			capi.ClusterLabelName: clusterID,
		}
	}

	err := c.List(ctx, machinePools, labelSelector, client.InNamespace(clusterNamespace))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return machinePools, nil
}

func GetOldExpMachinePoolsByMetadata(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*oldcapiexp.MachinePoolList, error) {
	if obj.Labels[capi.ClusterLabelName] == "" {
		err := microerror.Maskf(invalidObjectError, "Label %q must not be empty for object %q", capi.ClusterLabelName, obj.GetSelfLink())
		return nil, err
	}

	machinePools, err := GetOldExpMachinePoolsByClusterID(ctx, c, obj.Namespace, obj.Labels[capi.ClusterLabelName])
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return machinePools, nil
}

func GetOldExpMachinePoolsByClusterID(ctx context.Context, c client.Client, clusterNamespace, clusterID string) (*oldcapiexp.MachinePoolList, error) {
	machinePools := &oldcapiexp.MachinePoolList{}
	var labelSelector client.MatchingLabels
	{
		labelSelector = map[string]string{
			capi.ClusterLabelName: clusterID,
		}
	}

	err := c.List(ctx, machinePools, labelSelector, client.InNamespace(clusterNamespace))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return machinePools, nil
}

// GetAzureClusterFromMetadata returns the AzureCluster object (if present) using the object metadata.
func GetAzureClusterFromMetadata(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*capz.AzureCluster, error) {
	// Check if "cluster.x-k8s.io/cluster-name" label is set.
	if obj.Labels[capi.ClusterLabelName] == "" {
		err := microerror.Maskf(invalidObjectError, "Label %q must not be empty for object %q", capi.ClusterLabelName, obj.GetSelfLink())
		return nil, microerror.Mask(err)
	}

	return GetAzureClusterByName(ctx, c, obj.Namespace, obj.Labels[capi.ClusterLabelName])
}

// GetAzureClusterByName finds and return a AzureCluster object using the specified params.
func GetAzureClusterByName(ctx context.Context, c client.Client, namespace, name string) (*capz.AzureCluster, error) {
	azureCluster := &capz.AzureCluster{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.Get(ctx, key, azureCluster); err != nil {
		return nil, microerror.Mask(err)
	}

	return azureCluster, nil
}

// GetAzureMachinePoolByName finds and return a AzureMachinePool object using the specified params.
func GetAzureMachinePoolByName(ctx context.Context, c client.Client, namespace, name string) (*capzexp.AzureMachinePool, error) {
	azureMachinePool := &capzexp.AzureMachinePool{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.Get(ctx, key, azureMachinePool); err != nil {
		return nil, microerror.Mask(err)
	}

	return azureMachinePool, nil
}

func GetAzureMachinePoolsByMetadata(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*capzexp.AzureMachinePoolList, error) {
	if obj.Labels[capi.ClusterLabelName] == "" {
		err := microerror.Maskf(invalidObjectError, "Label %q must not be empty for object %q", capi.ClusterLabelName, obj.GetSelfLink())
		return nil, err
	}

	azureMachinePools, err := GetAzureMachinePoolsByClusterID(ctx, c, obj.Namespace, obj.Labels[capi.ClusterLabelName])
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
			capi.ClusterLabelName: clusterID,
		}
	}

	err := c.List(ctx, azureMachinePools, labelSelector, client.InNamespace(clusterNamespace))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureMachinePools, nil
}
