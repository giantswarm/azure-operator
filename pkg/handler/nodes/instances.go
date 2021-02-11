package nodes

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) GetVMSSInstances(ctx context.Context, azureMachinePool v1alpha3.AzureMachinePool) ([]compute.VirtualMachineScaleSetVM, error) {
	resourceGroupName := key.ClusterID(&azureMachinePool)
	vmssName := key.NodePoolVMSSName(&azureMachinePool)

	r.Logger.Debugf(ctx, "looking for the scale set %#q", vmssName)

	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, *azureCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	result, err := virtualMachineScaleSetVMsClient.List(ctx, resourceGroupName, vmssName, "", "", "")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var instances []compute.VirtualMachineScaleSetVM

	for result.NotDone() {
		instances = append(instances, result.Values()...)

		err := result.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r.Logger.Debugf(ctx, "found %d instances in the scale set %#q", len(instances), vmssName)

	return instances, nil
}

func (r *Resource) CreateARMDeployment(ctx context.Context, deploymentsClient *azureresource.DeploymentsClient, computedDeployment azureresource.Deployment, resourceGroupName, deploymentName string) error {
	res, err := deploymentsClient.CreateOrUpdate(ctx, resourceGroupName, deploymentName, computedDeployment)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = deploymentsClient.CreateOrUpdateResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
