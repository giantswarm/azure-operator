package vmsscheck

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type deleterJob struct {
	context                         context.Context
	logger                          micrologger.Logger
	virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient

	id                    string
	resourceGroup         string
	vmss                  string
	nextExecutionTime     time.Time
	allInstancesSucceeded bool

	onFinished func()
}

func (dj *deleterJob) ID() string {
	return dj.id
}

func (dj *deleterJob) Run() error {
	// Still not the time to run the check
	if !time.Now().After(dj.nextExecutionTime) {
		return nil
	}

	var err error
	dj.allInstancesSucceeded, err = dj.deleteFailedInstances(dj.context, dj.resourceGroup, dj.vmss)
	if IsNotFound(err) {
		// When resources are not found anymore, job can be considered to be completed.
		// This can happen when cluster is deleted in the middle of changes.
		dj.allInstancesSucceeded = true
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	dj.nextExecutionTime = time.Now().Add(10 * time.Second)
	return nil
}

func (dj *deleterJob) Finished() bool {
	// If any of the VMSS instances are in Failed state, return false here.
	if !dj.allInstancesSucceeded {
		return false
	}

	dj.onFinished()
	return true
}

// If any of the instances is not Succeeded, returns false.
// It deletes instances that are in "Failed" state.
func (dj *deleterJob) deleteFailedInstances(ctx context.Context, resourceGroup string, vmssName string) (bool, error) {
	// Get a list of instances in the VMSS.
	iterator, err := dj.virtualMachineScaleSetVMsClient.ListComplete(ctx, resourceGroup, vmssName, "", "", "")
	if err != nil {
		return false, microerror.Mask(err)
	}

	allSucceeded := true

	for iterator.NotDone() {
		instance := iterator.Value()

		dj.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Instance %s has state %s", *instance.Name, *instance.ProvisioningState))

		switch *instance.ProvisioningState {
		case provisioningStateFailed:
			// Reimage the instance.
			dj.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleting instance %s", *instance.Name))
			_, err := dj.virtualMachineScaleSetVMsClient.Delete(ctx, resourceGroup, vmssName, *instance.InstanceID)
			if err != nil {
				dj.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("Error deleting instance %s: %s", *instance.Name, err.Error()))
				return false, microerror.Mask(err)
			}

			dj.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Deleted instance %s", *instance.Name))
			allSucceeded = false
		case provisioningStateSucceeded:
			// OK to continue.
		default:
			// Just wait.
			allSucceeded = false
		}

		if err := iterator.NextWithContext(ctx); err != nil {
			return false, microerror.Mask(err)
		}
	}

	return allSucceeded, nil
}
