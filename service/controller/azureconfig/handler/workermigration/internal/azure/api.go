package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/client"
)

type api struct {
	wcAzureClientFactory client.CredentialsAwareClientFactoryInterface
	clusterID            string
}

func GetAPI(f client.CredentialsAwareClientFactoryInterface, clusterID string) API {
	return &api{
		wcAzureClientFactory: f,
		clusterID:            clusterID,
	}
}

func (a *api) GetVMSS(ctx context.Context, resourceGroupName, vmssName string) (VMSS, error) {
	client, err := a.wcAzureClientFactory.GetVirtualMachineScaleSetsClient(ctx, a.clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vmss, err := client.Get(ctx, resourceGroupName, vmssName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &vmss, nil
}

func (a *api) DeleteDeployment(ctx context.Context, resourceGroupName, deploymentName string) error {
	client, err := a.wcAzureClientFactory.GetDeploymentsClient(ctx, a.clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = client.Delete(ctx, resourceGroupName, deploymentName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (a *api) DeleteVMSS(ctx context.Context, resourceGroupName, vmssName string) error {
	client, err := a.wcAzureClientFactory.GetVirtualMachineScaleSetsClient(ctx, a.clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = client.Delete(ctx, resourceGroupName, vmssName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (a *api) ListVMSSNodes(ctx context.Context, resourceGroupName, vmssName string) (VMSSNodes, error) {
	client, err := a.wcAzureClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, a.clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	result, err := client.List(ctx, resourceGroupName, vmssName, "", "", "")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var vms []compute.VirtualMachineScaleSetVM
	for result.NotDone() {
		vms = append(vms, result.Values()...)

		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return vms, nil
}

func (a *api) ListNetworkSecurityGroups(ctx context.Context, resourceGroupName string) (SecurityGroups, error) {
	client, err := a.wcAzureClientFactory.GetNetworkSecurityGroupsClient(ctx, a.clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	result, err := client.List(ctx, resourceGroupName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var nsgs []network.SecurityGroup
	for result.NotDone() {
		nsgs = append(nsgs, result.Values()...)

		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return nsgs, nil
}

func (a *api) CreateOrUpdateNetworkSecurityGroup(ctx context.Context, resourceGroupName, networkSecurityGroupName string, securityGroup network.SecurityGroup) error {
	client, err := a.wcAzureClientFactory.GetNetworkSecurityGroupsClient(ctx, a.clusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = client.CreateOrUpdate(ctx, resourceGroupName, networkSecurityGroupName, securityGroup)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
