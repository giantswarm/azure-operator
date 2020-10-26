package vmsscheck

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	// Provisioning States.
	provisioningStateFailed    = "Failed"
	provisioningStateSucceeded = "Succeeded"
)

// Find out provisioning state of all VMSS instances and return true if all are
// Succeeded.
func InstancesAreRunning(ctx context.Context, logger micrologger.Logger, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, resourceGroup string, vmssName string) (bool, error) {
	// Get a list of instances in the VMSS.
	iterator, err := virtualMachineScaleSetVMsClient.ListComplete(ctx, resourceGroup, vmssName, "", "", "")
	if err != nil {
		return false, microerror.Mask(err)
	}

	allSucceeded := true

	for iterator.NotDone() {
		instance := iterator.Value()
		logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Instance %s has state %s", *instance.Name, *instance.ProvisioningState))

		switch *instance.ProvisioningState {
		case provisioningStateFailed:
			allSucceeded = false
		case provisioningStateSucceeded:
			// OK to continue.
		default:
			allSucceeded = false
		}

		if err := iterator.NextWithContext(ctx); err != nil {
			return false, microerror.Mask(err)
		}
	}

	return allSucceeded, nil
}
