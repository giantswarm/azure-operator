package vmsscheck

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
)

type InstanceWatchdog interface {
	GuardVMSS(ctx context.Context, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, resourceGroupName, vmssName string)
	DeleteFailedVMSS(ctx context.Context, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, resourceGroupName, vmssName string)
}
