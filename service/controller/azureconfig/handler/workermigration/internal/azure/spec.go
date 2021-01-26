package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
)

type VMSS *compute.VirtualMachineScaleSet
type VMSSNodes []compute.VirtualMachineScaleSetVM
type SecurityGroups []network.SecurityGroup

type API interface {
	// GetVMSS gets VMSS metadata from Azure API.
	GetVMSS(ctx context.Context, resourceGroupName, vmssName string) (VMSS, error)

	// DeleteDeployment deletes the corresponding deployment via Azure API.
	DeleteDeployment(ctx context.Context, resourceGroupName, deploymentName string) error

	// DeleteVMSS deletes the corresponding VMSS via Azure API.
	DeleteVMSS(ctx context.Context, resourceGroupName, vmssName string) error

	// ListVMSSNodes lists VMs in given VMSS via Azure API.
	ListVMSSNodes(ctx context.Context, resourceGroupName, vmssName string) (VMSSNodes, error)

	// ListNetworkSecurityGroups lists all network security groups in given resource group via Azure API.
	ListNetworkSecurityGroups(ctx context.Context, resourceGroupName string) (SecurityGroups, error)

	// CreateOrUpdateNetworkSecurityGroup creates or updates existing network security group via Azure API.
	CreateOrUpdateNetworkSecurityGroup(ctx context.Context, resourceGroupName, networkSecurityGroupName string, securityGroup network.SecurityGroup) error
}
