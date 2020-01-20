package instance

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"testing"
)

func Test_workingSet(t *testing.T) {
	var ws *workingSet

	if ws.IsWIP() != false {
		t.Fatal("<nil>.IsWIP() == true")
	}

	ws = ws.WithInstanceToUpdate(&compute.VirtualMachineScaleSetVM{})

	if ws.IsWIP() != true {
		t.Fatal("workingSet{...}.IsWIP() == false")
	}

	ws = nil

	ws = ws.WithInstanceToDrain(&compute.VirtualMachineScaleSetVM{})

	if ws.IsWIP() != true {
		t.Fatal("workingSet{...}.IsWIP() == false")
	}

	ws = nil

	ws = ws.WithInstanceToReimage(&compute.VirtualMachineScaleSetVM{})

	if ws.IsWIP() != true {
		t.Fatal("workingSet{...}.IsWIP() == false")
	}

	ws = nil

	ws = ws.WithInstanceAlreadyBeingUpdated(&compute.VirtualMachineScaleSetVM{})

	if ws.IsWIP() != true {
		t.Fatal("workingSet{...}.IsWIP() == false")
	}

	ws = nil
}
