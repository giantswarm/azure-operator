package migration

import (
	"context"
	"strconv"
	"testing"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"

	"github.com/giantswarm/azure-operator/v6/service/unittest"
)

func TestEnsureCreatedMachinePoolIsCorrect(t *testing.T) {
	testCases := []struct {
		name               string
		oldMachinePoolFile string
		newMachinePoolFile string
		expected           expected
	}{
		{
			name:               "case 0: New MachinePool references not updated",
			oldMachinePoolFile: "old_machinepool.yaml",
			newMachinePoolFile: "new_machinepool_references_not_updated.yaml",
		},
		{
			name:               "case 1: New MachinePool status not updated",
			oldMachinePoolFile: "old_machinepool.yaml",
			newMachinePoolFile: "new_machinepool_status_not_updated.yaml",
		},
		{
			name:               "case 2: New MachinePool fully created",
			oldMachinePoolFile: "old_machinepool.yaml",
			newMachinePoolFile: "new_machinepool_fully_created.yaml",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)
			ctx := context.Background()

			// Load old MachinePool
			oldMachinePool, err := loadOldMachinePoolCR(tc.oldMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			// Load new MachinePool, the one which would be created by machinepoolexp/migration handler
			newMachinePool, err := loadMachinePoolCR(tc.newMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			fakeK8sClient := unittest.FakeK8sClient(oldMachinePool, newMachinePool)

			// Create machinepool/migration handler
			var resource *Resource
			{
				c := Config{
					CtrlClient: fakeK8sClient.CtrlClient(),
					Logger:     microloggertest.New(),
				}
				resource, err = New(c)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Now run EnsureCreate for the new MachinePool
			err = resource.EnsureCreated(ctx, newMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			// Assert new MachinePool is correct
			namespacedName := types.NamespacedName{
				Namespace: newMachinePool.Namespace,
				Name:      newMachinePool.Name,
			}
			newMachinePoolAfterEnsureCreated := &capiexp.MachinePool{}
			verifyCR(t, fakeK8sClient.CtrlClient(), tc.name, newMachinePoolAfterEnsureCreated, namespacedName)
		})
	}
}

func TestEnsureCreatedMachinePoolUpdated(t *testing.T) {
	testCases := []struct {
		name               string
		oldMachinePoolFile string
		newMachinePoolFile string
		expected           expected
	}{
		{
			name:               "case 0: New MachinePool references not updated",
			oldMachinePoolFile: "old_machinepool.yaml",
			newMachinePoolFile: "new_machinepool_references_not_updated.yaml",
			expected: expected{
				newMachinePoolModificationsCount: 0,
			},
		},
		{
			name:               "case 1: New MachinePool status not updated",
			oldMachinePoolFile: "old_machinepool.yaml",
			newMachinePoolFile: "new_machinepool_status_not_updated.yaml",
			expected: expected{
				newMachinePoolModificationsCount: 1,
			},
		},
		{
			name:               "case 2: New MachinePool fully created",
			oldMachinePoolFile: "old_machinepool.yaml",
			newMachinePoolFile: "new_machinepool_fully_created.yaml",
			expected: expected{
				newMachinePoolModificationsCount: 0,
			},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)
			ctx := context.Background()

			// Load old MachinePool
			oldMachinePool, err := loadOldMachinePoolCR(tc.oldMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			// Load new MachinePool, the one which would be created by machinepoolexp/migration handler
			newMachinePool, err := loadMachinePoolCR(tc.newMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			fakeK8sClient := unittest.FakeK8sClient(oldMachinePool, newMachinePool)

			// Create machinepool/migration handler
			var resource *Resource
			{
				c := Config{
					CtrlClient: fakeK8sClient.CtrlClient(),
					Logger:     microloggertest.New(),
				}
				resource, err = New(c)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Now run EnsureCreate for the new MachinePool
			err = resource.EnsureCreated(ctx, newMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			// Assert: MachinePool modifications (check resource version to see
			// if there were updates)
			namespacedName := types.NamespacedName{
				Namespace: newMachinePool.Namespace,
				Name:      newMachinePool.Name,
			}
			newMachinePoolAfterEnsureCreated := &capiexp.MachinePool{}
			err = fakeK8sClient.CtrlClient().Get(ctx, namespacedName, newMachinePoolAfterEnsureCreated)
			if err != nil {
				t.Fatal(err)
			}

			// Get CR resource version before the update
			versionBeforeUpdate, err := strconv.Atoi(newMachinePool.ObjectMeta.ResourceVersion)
			if err != nil {
				t.Fatal(err)
			}

			// Get CR resource version after the update
			versionAfterUpdate, err := strconv.Atoi(newMachinePoolAfterEnsureCreated.ObjectMeta.ResourceVersion)
			if err != nil {
				t.Fatal(err)
			}

			if versionAfterUpdate != versionBeforeUpdate+tc.expected.newMachinePoolModificationsCount {
				t.Fatalf(
					"expected %d new MachinePool modification(s), got %d",
					tc.expected.newMachinePoolModificationsCount,
					versionAfterUpdate-versionBeforeUpdate)
			}
		})
	}
}

func TestEnsureCreatedOldMachinePoolDeleted(t *testing.T) {
	testCases := []struct {
		name               string
		oldMachinePoolFile string
		newMachinePoolFile string
		expected           expected
	}{
		{
			name:               "case 0: New MachinePool references not updated",
			oldMachinePoolFile: "old_machinepool.yaml",
			newMachinePoolFile: "new_machinepool_references_not_updated.yaml",
			expected: expected{
				oldMachinePoolDeleted: false,
			},
		},
		{
			name:               "case 1: New MachinePool status not updated",
			oldMachinePoolFile: "old_machinepool.yaml",
			newMachinePoolFile: "new_machinepool_status_not_updated.yaml",
			expected: expected{
				oldMachinePoolDeleted: true,
			},
		},
		{
			name:               "case 2: New MachinePool fully created",
			oldMachinePoolFile: "old_machinepool.yaml",
			newMachinePoolFile: "new_machinepool_fully_created.yaml",
			expected: expected{
				oldMachinePoolDeleted: true,
			},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)
			ctx := context.Background()

			// Load old MachinePool
			oldMachinePool, err := loadOldMachinePoolCR(tc.oldMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			// Load new MachinePool, the one which would be created by machinepoolexp/migration handler
			newMachinePool, err := loadMachinePoolCR(tc.newMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			fakeK8sClient := unittest.FakeK8sClient(oldMachinePool, newMachinePool)

			// Create machinepool/migration handler
			var resource *Resource
			{
				c := Config{
					CtrlClient: fakeK8sClient.CtrlClient(),
					Logger:     microloggertest.New(),
				}
				resource, err = New(c)
				if err != nil {
					t.Fatal(err)
				}
			}

			// Now run EnsureCreate for the new MachinePool
			err = resource.EnsureCreated(ctx, newMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			// Assert: old MachinePool exists or not
			namespacedName := types.NamespacedName{
				Namespace: newMachinePool.Namespace,
				Name:      newMachinePool.Name,
			}
			oldMachinePool = &oldcapiexpv1alpha3.MachinePool{}
			err = fakeK8sClient.CtrlClient().Get(ctx, namespacedName, oldMachinePool)

			oldMachinePoolDeleted := false
			if err != nil {
				if errors.IsNotFound(err) {
					oldMachinePoolDeleted = true
				} else {
					t.Fatalf("unexpected error %#q", err)
				}
			}
			// if the CR has finalizers, fakeClient delete will just set DeletionTimestamp
			if !oldMachinePoolDeleted && oldMachinePool.ObjectMeta.DeletionTimestamp != nil {
				oldMachinePoolDeleted = true
			}

			if tc.expected.oldMachinePoolDeleted && !oldMachinePoolDeleted {
				t.Fatalf("expected that old MachinePool is deleted, but it found it: %+v", oldMachinePool)
			} else if !tc.expected.oldMachinePoolDeleted && oldMachinePoolDeleted {
				t.Fatalf("expected that old MachinePool is not deleted, but it is deleted")
			}
		})
	}
}
