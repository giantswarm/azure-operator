package migration

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capiexp/v1alpha3"
	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/apimachinery/pkg/types"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
)

func TestEnsureCreatedCreatesNewMachinePool(t *testing.T) {
	testCases := []struct {
		name                    string
		oldMachinePoolFile      string
		oldAzureMachinePoolFile string
	}{
		{
			name:                    "case 0: Simple MachinePool",
			oldMachinePoolFile:      "old_machinepool.yaml",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			fakeClient := newFakeClient()
			ctx := context.Background()

			// Get old exp MachinePool from test file, this is the CR that the
			// handler is reconciling.
			o, err := loadCR(tc.oldMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}
			oldExpMachinePool, ok := o.(*oldcapiexpv1alpha3.MachinePool)
			if !ok {
				t.Fatalf("couldn't cast object %T to %T", o, oldExpMachinePool)
			}

			// Get old exp AzureMachinePool from test file, this CR should exist in etcd.
			o, err = loadCR(tc.oldAzureMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}
			oldExpAzureMachinePool, ok := o.(*oldcapzexpv1alpha3.AzureMachinePool)
			if !ok {
				t.Fatalf("couldn't cast object %T to %T", o, oldExpMachinePool)
			}
			err = fakeClient.Create(ctx, oldExpAzureMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			var resource *Resource
			{
				c := Config{
					CtrlClient: fakeClient,
					Logger:     microloggertest.New(),
				}
				resource, err = New(c)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Run MachinePool migration
			err = resource.EnsureCreated(ctx, oldExpMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			// Now verify that we have new MachinePool CR created
			newMachinePoolName := types.NamespacedName{
				Namespace: oldExpMachinePool.Namespace,
				Name:      oldExpMachinePool.Name,
			}

			verifyCR(t, fakeClient, tc.name, new(capiexp.MachinePool), newMachinePoolName)
		})
	}
}

func TestEnsureCreatedDoesNotSetMachinePoolStatus(t *testing.T) {
	testCases := []struct {
		name                    string
		oldMachinePoolFile      string
		oldAzureMachinePoolFile string
	}{
		{
			name:                    "case 0: Simple MachinePool",
			oldMachinePoolFile:      "old_machinepool.yaml",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)

			fakeClient := newFakeClient()
			ctx := context.Background()

			// Get old exp MachinePool from test file, this is the CR that the
			// handler is reconciling.
			o, err := loadCR(tc.oldMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}
			oldExpMachinePool, ok := o.(*oldcapiexpv1alpha3.MachinePool)
			if !ok {
				t.Fatalf("couldn't cast object %T to %T", o, oldExpMachinePool)
			}

			// Get old exp AzureMachinePool from test file, this CR should exist in etcd.
			o, err = loadCR(tc.oldAzureMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}
			oldExpAzureMachinePool, ok := o.(*oldcapzexpv1alpha3.AzureMachinePool)
			if !ok {
				t.Fatalf("couldn't cast object %T to %T", o, oldExpMachinePool)
			}
			err = fakeClient.Create(ctx, oldExpAzureMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			var resource *Resource
			{
				c := Config{
					CtrlClient: fakeClient,
					Logger:     microloggertest.New(),
				}
				resource, err = New(c)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Run MachinePool migration
			err = resource.EnsureCreated(ctx, oldExpMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			// Check Status in the new MachinePool
			// Now verify that we have new MachinePool CR created
			namespacedName := types.NamespacedName{
				Namespace: oldExpMachinePool.Namespace,
				Name:      oldExpMachinePool.Name,
			}
			newMachinePool := capiexp.MachinePool{}
			err = fakeClient.Get(context.Background(), namespacedName, &newMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(newMachinePool.Status, capiexp.MachinePoolStatus{}) {
				t.Fatalf("migration handler must not set Status field in the new MachinePool")
			}
		})
	}
}
