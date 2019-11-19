package instance

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
)

type workingSet struct {
	instanceToUpdate            *compute.VirtualMachineScaleSetVM
	instanceToDrain             *compute.VirtualMachineScaleSetVM
	instanceToReimage           *compute.VirtualMachineScaleSetVM
	instanceAlreadyBeingUpdated *compute.VirtualMachineScaleSetVM
}

func (ws *workingSet) WithInstanceToUpdate(instance *compute.VirtualMachineScaleSetVM) *workingSet {
	if ws == nil {
		ws = &workingSet{}
	}
	ws.instanceToUpdate = instance
	return ws
}

func (ws *workingSet) WithInstanceToDrain(instance *compute.VirtualMachineScaleSetVM) *workingSet {
	if ws == nil {
		ws = &workingSet{}
	}
	ws.instanceToDrain = instance
	return ws
}

func (ws *workingSet) WithInstanceToReimage(instance *compute.VirtualMachineScaleSetVM) *workingSet {
	if ws == nil {
		ws = &workingSet{}
	}
	ws.instanceToReimage = instance
	return ws
}

func (ws *workingSet) WithInstanceAlreadyBeingUpdated(instance *compute.VirtualMachineScaleSetVM) *workingSet {
	if ws == nil {
		ws = &workingSet{}
	}
	ws.instanceAlreadyBeingUpdated = instance
	return ws
}

func (ws *workingSet) IsWIP() bool {
	if ws == nil {
		return false
	}

	return ws.instanceToUpdate != nil ||
		ws.instanceToDrain != nil ||
		ws.instanceToReimage != nil ||
		ws.instanceAlreadyBeingUpdated != nil
}

func (ws *workingSet) InstanceToUpdate() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceToUpdate
}

func (ws *workingSet) InstanceToDrain() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceToDrain
}

func (ws *workingSet) InstanceToReimage() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceToReimage
}

func (ws *workingSet) InstanceAlreadyBeingUpdated() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceAlreadyBeingUpdated
}

// getWorkingSet either returns an instance to update or an instance to
// reimage, but never both at the same time.
func getWorkingSet(customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM, drainerConfigs []corev1alpha1.DrainerConfig, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error), versionValue map[string]string) (*workingSet, error) {
	var err error

	var ws *workingSet

	instanceInProgress := firstInstanceInProgress(customObject, instances)
	if instanceInProgress != nil {
		return ws.WithInstanceAlreadyBeingUpdated(instanceInProgress), nil
	}

	var instanceToUpdate *compute.VirtualMachineScaleSetVM
	instanceToUpdate = firstInstanceToUpdate(customObject, instances)
	if instanceToUpdate != nil {
		return ws.WithInstanceToUpdate(instanceToUpdate), nil
	}

	var instanceToReimage *compute.VirtualMachineScaleSetVM
	instanceToReimage, err = firstInstanceToReimage(customObject, instances, instanceNameFunc, versionValue)
	if err != nil {
		return ws, microerror.Mask(err)
	}
	if instanceToReimage != nil {
		instanceName, err := instanceNameFunc(customObject, *instanceToReimage.InstanceID)
		if err != nil {
			return ws, microerror.Mask(err)
		}
		if isNodeDrained(drainerConfigs, instanceName) {
			return ws.WithInstanceToReimage(instanceToReimage), nil
		} else {
			return ws.WithInstanceToDrain(instanceToReimage), nil
		}
	}

	return ws, nil
}
