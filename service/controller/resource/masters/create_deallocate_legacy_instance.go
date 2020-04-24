package masters

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) deallocateLegacyInstanceTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	deallocated, err := r.isVMSSInstanceDeallocated(ctx, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
	// TODO Check for not found error because it's ok to continue in that case.
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	if !deallocated {
		r.logger.LogCtx(ctx, "level", "info", "message", "Legacy VMSS instance is not deallocated yet.")
		r.logger.LogCtx(ctx, "level", "info", "message", "Deallocating legacy VMSS instances.")
		err := r.deallocateAllInstances(ctx, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
		if err != nil {
			return Empty, microerror.Mask(err)
		}
		r.logger.LogCtx(ctx, "level", "info", "message", "Deallocated legacy VMSS instances.")
		return currentState, nil
	}

	return BlockAPICalls, nil
}

func (r *Resource) deallocateAllInstances(ctx context.Context, resourceGroup string, vmssName string) error {
	vmssInstancesClient, err := r.getVMsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	instancesRunning, err := r.getRunningInstances(ctx, resourceGroup, vmssName)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(instancesRunning) > 0 {
		// There are instances still not deallocated.
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("There are %d instances to be deallocated.", len(instancesRunning)))

		for _, instance := range instancesRunning {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Requesting Deallocate for %s", *instance.Name))
			_, err = vmssInstancesClient.Deallocate(ctx, resourceGroup, vmssName, *instance.InstanceID)
			if err != nil {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Error requesting Deallocate for %s: %s", *instance.Name, err.Error()))
				continue
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Requested Deallocate for %s", *instance.Name))
		}
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "All instances are deallocated.")
	}

	return nil
}

func (r *Resource) getRunningInstances(ctx context.Context, resourceGroup string, vmssName string) ([]compute.VirtualMachineScaleSetVM, error) {
	vmssInstancesClient, err := r.getVMsClient(ctx)
	if err != nil {
		return []compute.VirtualMachineScaleSetVM{}, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Iterating on the %s instances to find any instance still running", vmssName))

	result, err := vmssInstancesClient.List(ctx, resourceGroup, vmssName, "", "", "")
	if err != nil {
		return []compute.VirtualMachineScaleSetVM{}, microerror.Mask(err)
	}

	var instancesRunning []compute.VirtualMachineScaleSetVM

	// The List response doesn't contain the PowerState data for the instances.
	// We need to call GetInstanceView on every instance to get such information.
	for result.NotDone() {
		for _, instance := range result.Values() {
			details, err := vmssInstancesClient.GetInstanceView(context.Background(), resourceGroup, vmssName, *instance.InstanceID)
			if err != nil {
				return []compute.VirtualMachineScaleSetVM{}, microerror.Mask(err)
			}

			for _, instanceState := range *details.Statuses {
				// TODO move strings elsewhere
				if strings.HasPrefix(*instanceState.Code, "PowerState/") && *instanceState.Code != "PowerState/deallocated" {
					r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Instance %s is in status %s", *instance.Name, *instanceState.Code))
					// Machine is not deallocated.
					instancesRunning = append(instancesRunning, instance)
					continue
				}
			}
		}

		err := result.Next()
		if err != nil {
			return []compute.VirtualMachineScaleSetVM{}, microerror.Mask(err)
		}
	}

	return instancesRunning, nil
}

func (r *Resource) isVMSSInstanceDeallocated(ctx context.Context, resourceGroup string, vmssName string) (bool, error) {
	instancesRunning, err := r.getRunningInstances(ctx, resourceGroup, vmssName)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return len(instancesRunning) == 0, nil
}
