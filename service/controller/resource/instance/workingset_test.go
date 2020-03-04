package instance

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
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
