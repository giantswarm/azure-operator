package masters

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	PowerStateLabelPrefix = "PowerState/"
	PowerStateDeallocated = "PowerState/deallocated"
)

func (r *Resource) deallocateLegacyInstanceTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	deallocated, err := r.isVMSSInstanceDeallocated(ctx, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
	if IsNotFound(err) {
		return BlockAPICalls, nil
	} else if err != nil {
		return Empty, microerror.Mask(err)
	}

	if !deallocated {
		r.Logger.LogCtx(ctx, "level", "info", "message", "Legacy VMSS instance is not deallocated yet.")
		r.Logger.LogCtx(ctx, "level", "info", "message", "Deallocating legacy VMSS instances.")
		err := r.deallocateAllInstances(ctx, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
		if err != nil {
			return Empty, microerror.Mask(err)
		}
		r.Logger.LogCtx(ctx, "level", "info", "message", "Deallocated legacy VMSS instances.")
		return currentState, nil
	}

	return BlockAPICalls, nil
}

func (r *Resource) deallocateAllInstances(ctx context.Context, resourceGroup string, vmssName string) error {
	vmssInstancesClient, err := r.GetVMsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	instancesRunning, err := r.getRunningInstances(ctx, resourceGroup, vmssName)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(instancesRunning) > 0 {
		// There are instances still not deallocated.
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("There are %d instances to be deallocated.", len(instancesRunning)))

		for _, instance := range instancesRunning {
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Requesting Deallocate for %s", *instance.Name))
			_, err = vmssInstancesClient.Deallocate(ctx, resourceGroup, vmssName, *instance.InstanceID)
			if err != nil {
				r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Error requesting Deallocate for %s: %s", *instance.Name, err.Error()))
				continue
			}
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Requested Deallocate for %s", *instance.Name))
		}
	} else {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "All instances are deallocated.")
	}

	return nil
}

func (r *Resource) getRunningInstances(ctx context.Context, resourceGroup string, vmssName string) ([]compute.VirtualMachineScaleSetVM, error) {
	vmssInstancesClient, err := r.GetVMsClient(ctx)
	if err != nil {
		return []compute.VirtualMachineScaleSetVM{}, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Iterating on the %s instances to find any instance still running", vmssName))

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
				if strings.HasPrefix(*instanceState.Code, PowerStateLabelPrefix) && *instanceState.Code != PowerStateDeallocated {
					r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Instance %s is in status %s", *instance.Name, *instanceState.Code))
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

func (r *Resource) getVMSS(ctx context.Context, resourceGroup string, vmssName string) (*compute.VirtualMachineScaleSet, error) {
	c, err := r.GetScaleSetsClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vmss, err := c.Get(ctx, resourceGroup, vmssName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &vmss, nil
}

func (r *Resource) isVMSSInstanceDeallocated(ctx context.Context, resourceGroup string, vmssName string) (bool, error) {
	instancesRunning, err := r.getRunningInstances(ctx, resourceGroup, vmssName)
	if err != nil {
		return false, microerror.Mask(err)
	}

	return len(instancesRunning) == 0, nil
}
