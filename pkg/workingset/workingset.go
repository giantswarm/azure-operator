package workingset

import "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"

type WorkingSet struct {
	instanceToUpdate            *compute.VirtualMachineScaleSetVM
	instanceToDrain             *compute.VirtualMachineScaleSetVM
	instanceToReimage           *compute.VirtualMachineScaleSetVM
	instanceAlreadyBeingUpdated *compute.VirtualMachineScaleSetVM
}

func (ws *WorkingSet) WithInstanceToUpdate(instance *compute.VirtualMachineScaleSetVM) *WorkingSet {
	if ws == nil {
		ws = &WorkingSet{}
	}
	ws.instanceToUpdate = instance
	return ws
}

func (ws *WorkingSet) WithInstanceToDrain(instance *compute.VirtualMachineScaleSetVM) *WorkingSet {
	if ws == nil {
		ws = &WorkingSet{}
	}
	ws.instanceToDrain = instance
	return ws
}

func (ws *WorkingSet) WithInstanceToReimage(instance *compute.VirtualMachineScaleSetVM) *WorkingSet {
	if ws == nil {
		ws = &WorkingSet{}
	}
	ws.instanceToReimage = instance
	return ws
}

func (ws *WorkingSet) WithInstanceAlreadyBeingUpdated(instance *compute.VirtualMachineScaleSetVM) *WorkingSet {
	if ws == nil {
		ws = &WorkingSet{}
	}
	ws.instanceAlreadyBeingUpdated = instance
	return ws
}

func (ws *WorkingSet) IsWIP() bool {
	if ws == nil {
		return false
	}

	return ws.instanceToUpdate != nil ||
		ws.instanceToDrain != nil ||
		ws.instanceToReimage != nil ||
		ws.instanceAlreadyBeingUpdated != nil
}

func (ws *WorkingSet) InstanceToUpdate() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceToUpdate
}

func (ws *WorkingSet) InstanceToDrain() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceToDrain
}

func (ws *WorkingSet) InstanceToReimage() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceToReimage
}

func (ws *WorkingSet) InstanceAlreadyBeingUpdated() *compute.VirtualMachineScaleSetVM {
	if ws == nil {
		return nil
	}

	return ws.instanceAlreadyBeingUpdated
}
