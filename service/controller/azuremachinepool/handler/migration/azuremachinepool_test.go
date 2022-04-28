package migration

import (
	"context"
	"strconv"
	"testing"

	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/micrologger/microloggertest"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"

	"github.com/giantswarm/azure-operator/v5/service/unittest"
)

func TestEnsureCreatedAzureMachinePoolIsCorrect(t *testing.T) {
	testCases := []struct {
		name                    string
		oldAzureMachinePoolFile string
		newAzureMachinePoolFile string
		expected                expected
	}{
		{
			name:                    "case 0: New AzureMachinePool references not updated",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
			newAzureMachinePoolFile: "new_azuremachinepool_references_not_updated.yaml",
		},
		{
			name:                    "case 1: New AzureMachinePool status not updated",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
			newAzureMachinePoolFile: "new_azuremachinepool_status_not_updated.yaml",
		},
		{
			name:                    "case 2: New AzureMachinePool fully created",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
			newAzureMachinePoolFile: "new_azuremachinepool_fully_created.yaml",
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)
			ctx := context.Background()

			// Load old AzureMachinePool
			oldAzureMachinePool, err := loadOldAzureMachinePoolCR(tc.oldAzureMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			// Load new AzureMachinePool, the one which would be created by machinepoolexp/migration handler
			newAzureMachinePool, err := loadAzureMachinePoolCR(tc.newAzureMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			fakeK8sClient := unittest.FakeK8sClient(oldAzureMachinePool, newAzureMachinePool)

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

			// Now run EnsureCreate for the new AzureMachinePool
			err = resource.EnsureCreated(ctx, newAzureMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			// Assert: New AzureMachinePool is correct
			namespacedName := types.NamespacedName{
				Namespace: newAzureMachinePool.Namespace,
				Name:      newAzureMachinePool.Name,
			}
			newAzureMachinePoolAfterEnsureCreated := &capzexp.AzureMachinePool{}
			verifyCR(t, fakeK8sClient.CtrlClient(), tc.name, newAzureMachinePoolAfterEnsureCreated, namespacedName)
		})
	}
}

func TestEnsureCreatedAzureMachinePoolUpdated(t *testing.T) {
	testCases := []struct {
		name                    string
		oldAzureMachinePoolFile string
		newAzureMachinePoolFile string
		expected                expected
	}{
		{
			name:                    "case 0: New AzureMachinePool references not updated",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
			newAzureMachinePoolFile: "new_azuremachinepool_references_not_updated.yaml",
			expected: expected{
				newAzureMachinePoolModificationsCount: 0,
			},
		},
		{
			name:                    "case 1: New AzureMachinePool status not updated",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
			newAzureMachinePoolFile: "new_azuremachinepool_status_not_updated.yaml",
			expected: expected{
				newAzureMachinePoolModificationsCount: 1,
			},
		},
		{
			name:                    "case 2: New AzureMachinePool fully created",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
			newAzureMachinePoolFile: "new_azuremachinepool_fully_created.yaml",
			expected: expected{
				newAzureMachinePoolModificationsCount: 0,
			},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)
			ctx := context.Background()

			// Load old AzureMachinePool
			oldAzureMachinePool, err := loadOldAzureMachinePoolCR(tc.oldAzureMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			// Load new AzureMachinePool, the one which would be created by machinepoolexp/migration handler
			newAzureMachinePool, err := loadAzureMachinePoolCR(tc.newAzureMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			fakeK8sClient := unittest.FakeK8sClient(oldAzureMachinePool, newAzureMachinePool)

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

			// Now run EnsureCreate for the new AzureMachinePool
			err = resource.EnsureCreated(ctx, newAzureMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			// Assert: AzureMachinePool modifications (check resource version to see
			// if there were updates)
			namespacedName := types.NamespacedName{
				Namespace: newAzureMachinePool.Namespace,
				Name:      newAzureMachinePool.Name,
			}
			newAzureMachinePoolAfterEnsureCreated := &capzexp.AzureMachinePool{}
			err = fakeK8sClient.CtrlClient().Get(ctx, namespacedName, newAzureMachinePoolAfterEnsureCreated)
			if err != nil {
				t.Fatal(err)
			}

			// Get CR resource version before the update
			versionBeforeUpdate, err := strconv.Atoi(newAzureMachinePool.ObjectMeta.ResourceVersion)
			if err != nil {
				t.Fatal(err)
			}

			// Get CR resource version after the update
			versionAfterUpdate, err := strconv.Atoi(newAzureMachinePoolAfterEnsureCreated.ObjectMeta.ResourceVersion)
			if err != nil {
				t.Fatal(err)
			}

			if versionAfterUpdate != versionBeforeUpdate+tc.expected.newAzureMachinePoolModificationsCount {
				t.Fatalf(
					"expected %d new AzureMachinePool modification(s), got %d",
					tc.expected.newAzureMachinePoolModificationsCount,
					versionAfterUpdate-versionBeforeUpdate)
			}
		})
	}
}

func TestEnsureCreatedOldAzureMachinePoolDeleted(t *testing.T) {
	testCases := []struct {
		name                    string
		oldAzureMachinePoolFile string
		newAzureMachinePoolFile string
		expected                expected
	}{
		{
			name:                    "case 0: New AzureMachinePool references not updated",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
			newAzureMachinePoolFile: "new_azuremachinepool_references_not_updated.yaml",
			expected: expected{
				oldAzureMachinePoolDeleted: false,
			},
		},
		{
			name:                    "case 1: New AzureMachinePool status not updated",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
			newAzureMachinePoolFile: "new_azuremachinepool_status_not_updated.yaml",
			expected: expected{
				oldAzureMachinePoolDeleted: true,
			},
		},
		{
			name:                    "case 2: New AzureMachinePool fully created",
			oldAzureMachinePoolFile: "old_azuremachinepool.yaml",
			newAzureMachinePoolFile: "new_azuremachinepool_fully_created.yaml",
			expected: expected{
				oldAzureMachinePoolDeleted: true,
			},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Log(tc.name)
			ctx := context.Background()

			// Load old AzureMachinePool
			oldAzureMachinePool, err := loadOldAzureMachinePoolCR(tc.oldAzureMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			// Load new AzureMachinePool, the one which would be created by machinepoolexp/migration handler
			newAzureMachinePool, err := loadAzureMachinePoolCR(tc.newAzureMachinePoolFile)
			if err != nil {
				t.Fatal(err)
			}

			fakeK8sClient := unittest.FakeK8sClient(oldAzureMachinePool, newAzureMachinePool)

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

			// Now run EnsureCreate for the new AzureMachinePool
			err = resource.EnsureCreated(ctx, newAzureMachinePool)
			if err != nil {
				t.Fatal(err)
			}

			// Assert: old AzureMachinePool exists or not
			namespacedName := types.NamespacedName{
				Namespace: newAzureMachinePool.Namespace,
				Name:      newAzureMachinePool.Name,
			}
			newAzureMachinePoolAfterEnsureCreated := &capzexp.AzureMachinePool{}
			err = fakeK8sClient.CtrlClient().Get(ctx, namespacedName, newAzureMachinePoolAfterEnsureCreated)
			if err != nil {
				t.Fatal(err)
			}

			oldAzureMachinePool = &oldcapzexpv1alpha3.AzureMachinePool{}
			err = fakeK8sClient.CtrlClient().Get(ctx, namespacedName, oldAzureMachinePool)

			oldAzureMachinePoolDeleted := false
			if err != nil {
				if errors.IsNotFound(err) {
					oldAzureMachinePoolDeleted = true
				} else {
					t.Fatalf("unexpected error %#q", err)
				}
			}

			if tc.expected.oldAzureMachinePoolDeleted && !oldAzureMachinePoolDeleted {
				t.Fatalf("expected that old AzureMachinePool is deleted, but it found it: %+v", oldAzureMachinePool)
			} else if !tc.expected.oldAzureMachinePoolDeleted && oldAzureMachinePoolDeleted {
				t.Fatalf("expected that old AzureMachinePool is not deleted, but it is deleted")
			}
		})
	}
}
