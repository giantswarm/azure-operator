package instance

import "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"

type workingSet struct {
	instanceToUpdate            *compute.VirtualMachineScaleSetVM
	instanceToDrain             *compute.VirtualMachineScaleSetVM
	instanceToReimage           *compute.VirtualMachineScaleSetVM
	instanceAlreadyBeingUpdated *compute.VirtualMachineScaleSetVM
}

func (ws *workingSet) isWIP() bool {
	return ws.instanceToUpdate != nil ||
		ws.instanceToDrain != nil ||
		ws.instanceToReimage != nil ||
		ws.instanceAlreadyBeingUpdated != nil
}

func newWorkingSetEmpty() workingSet {
	return workingSet{}
}

func newWorkingSetFromInstanceToUpdate(vm *compute.VirtualMachineScaleSetVM) workingSet {
	return workingSet{
		instanceToUpdate: vm,
	}
}

func newWorkingSetFromInstanceToDrain(vm *compute.VirtualMachineScaleSetVM) workingSet {
	return workingSet{
		instanceToDrain: vm,
	}
}

func newWorkingSetFromInstanceToReimage(vm *compute.VirtualMachineScaleSetVM) workingSet {
	return workingSet{
		instanceToReimage: vm,
	}
}

func newWorkingSetFromInstanceAlreadyBeingUpdated(vm *compute.VirtualMachineScaleSetVM) workingSet {
	return workingSet{
		instanceAlreadyBeingUpdated: vm,
	}
}
