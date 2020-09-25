package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
)

type VMSS *compute.VirtualMachineScaleSet
type VMSSNodes []compute.VirtualMachineScaleSetVM

type API interface {
	// GetVMSS gets VMSS metadata from Azure API.
	GetVMSS(ctx context.Context, resourceGroupName, vmssName string) (VMSS, error)

	// DeleteVMSS deletes the corresponding VMSS via Azure API.
	DeleteVMSS(ctx context.Context, resourceGroupName, vmssName string) error

	// ListVMSSNodes lists VMs in given VMSS via Azure API.
	ListVMSSNodes(ctx context.Context, resourceGroupName, vmssName string) (VMSSNodes, error)
}
