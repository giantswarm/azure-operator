package nodepool

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// EnsureDeleted is a noop since the deletion of deployments is redirected to
// the deletion of resource groups because they garbage collect them.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	azureCluster, err := r.getAzureClusterFromCluster(ctx, cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	tenantClusterK8sClient, err := r.tenantClientFactory.GetClient(ctx, cluster)
	if tenantcluster.IsAPINotAvailableError(err) {
		r.Logger.Debugf(ctx, "tenant API not available yet")
		r.Logger.Debugf(ctx, "canceling resource")

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	err = r.removeNodesFromK8s(ctx, tenantClusterK8sClient, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.removeNodePool(ctx, &azureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.removeSubnetFromAzureCluster(ctx, azureCluster, azureMachinePool.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) removeSubnetFromAzureCluster(ctx context.Context, azureCluster *capzv1alpha3.AzureCluster, subnetName string) error {
	subnetPosition := -1
	for i, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		if subnet.Name == subnetName {
			subnetPosition = i
		}
	}

	subnetIsFound := subnetPosition >= 0
	if subnetIsFound {
		azureCluster.Spec.NetworkSpec.Subnets = append(azureCluster.Spec.NetworkSpec.Subnets[:subnetPosition], azureCluster.Spec.NetworkSpec.Subnets[subnetPosition+1:]...)

		r.Logger.LogCtx(ctx, "message", "Ensuring subnet is not in AzureCluster", "subnetName", subnetName)

		err := r.CtrlClient.Update(ctx, azureCluster)
		if err != nil {
			return microerror.Mask(err)
		}

		r.Logger.LogCtx(ctx, "message", "Ensured subnet is not in AzureCluster", "subnetName", subnetName)
	}

	return nil
}

func (r *Resource) removeNodePool(ctx context.Context, azureMachinePool *capzexpv1alpha3.AzureMachinePool) error {
	var err error

	err = r.deleteARMDeployment(ctx, azureMachinePool, key.ClusterID(azureMachinePool), key.NodePoolDeploymentName(azureMachinePool))
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.deleteVMSS(ctx, azureMachinePool, key.ClusterID(azureMachinePool), key.NodePoolVMSSName(azureMachinePool))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Deletes all the node objects belonging to the node pool using the k8s API.
// This happens automatically eventually, but we make this much quicker by doing it on the API server directly.
func (r *Resource) removeNodesFromK8s(ctx context.Context, ctrlClient client.Client, azureMachinePool *capzexpv1alpha3.AzureMachinePool) error {
	r.Logger.LogCtx(ctx, "message", fmt.Sprintf("Deleting nodes from k8s API for machine pool %s", azureMachinePool.Name))

	err := ctrlClient.DeleteAllOf(ctx, &corev1.Node{}, client.MatchingLabels{label.MachinePool: azureMachinePool.Name})
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "message", fmt.Sprintf("Deleted nodes from k8s API for machine pool %s", azureMachinePool.Name))

	return nil
}

// deleteARMDeployment deletes the ARM deployment from Azure.
func (r *Resource) deleteARMDeployment(ctx context.Context, azureMachinePool *capzexpv1alpha3.AzureMachinePool, resourceGroupName, deploymentName string) error {
	r.Logger.Debugf(ctx, "Deleting machine pool ARM deployment")

	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(ctx, key.ClusterID(azureMachinePool))
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = deploymentsClient.Delete(ctx, resourceGroupName, deploymentName)
	if IsDeploymentNotFound(err) {
		r.Logger.Debugf(ctx, "Machine pool ARM deployment was already deleted")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.Debugf(ctx, "Deleted machine pool ARM deployment")

	return nil
}

// deleteVMSS deletes the VMSS from Azure.
func (r *Resource) deleteVMSS(ctx context.Context, azureMachinePool *capzexpv1alpha3.AzureMachinePool, resourceGroupName, vmssName string) error {
	r.Logger.LogCtx(ctx, "message", "Deleting machine pool VMSS")

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, key.ClusterID(azureMachinePool))
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = virtualMachineScaleSetsClient.Delete(ctx, resourceGroupName, vmssName)
	if IsNotFound(err) {
		r.Logger.LogCtx(ctx, "message", "Machine pool VMSS was already deleted")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "message", "Deleted machine pool VMSS")

	return nil
}
