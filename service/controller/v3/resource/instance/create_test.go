package instance

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

func Test_Resource_Instance_findActionableInstance(t *testing.T) {
	testCases := []struct {
		Name                      string
		CustomObject              providerv1alpha1.AzureConfig
		Instances                 []compute.VirtualMachineScaleSetVM
		VersionValue              map[string]string
		ExpectedInstanceToUpdate  *compute.VirtualMachineScaleSetVM
		ExpectedInstanceToReimage *compute.VirtualMachineScaleSetVM
		ErrorMatcher              func(err error) bool
	}{
		{
			Name:                      "case 0: empty input results in no action",
			CustomObject:              providerv1alpha1.AzureConfig{},
			Instances:                 nil,
			VersionValue:              nil,
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              IsVersionBlobEmpty,
		},
		{
			Name: "case 1: one instance being up to date results in no action",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 2: two instances being up to date results in no action",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.1.0",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 3: one instance not having the latest model applied results in updating the instance",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("alq9y-worker-000001"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 4: two instances not having the latest model applied results in updating the first instance",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.1.0",
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("alq9y-worker-000001"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 5: one instance having the latest model applied and one instance not having the latest model applied results in updating the second instance",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.1.0",
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("alq9y-worker-000002"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 6: two instances not having the latest version bundle version applied results in reimaging the first instance",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.0.1",
				"alq9y-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("alq9y-worker-000001"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 7: one instance having the latest version bundle version applied and one instance not having the latest version bundle version applied results in reimaging the second instance",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("alq9y-worker-000002"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 8: two instances not having the latest version bundle version applied and being in provisioning state 'InProgress' results in no action",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
					},
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
					},
				},
			},
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.0.1",
				"alq9y-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 9: one instance having the latest version bundle version applied and one instance not having the latest version bundle version applied and being in provisioning state 'InProgress' results in no action",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
					},
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
					},
				},
			},
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			instanceToUpdate, instanceToReimage, err := findActionableInstance(tc.CustomObject, tc.Instances, tc.VersionValue)

			switch {
			case err == nil && tc.ErrorMatcher == nil:
				// fall through
			case err != nil && tc.ErrorMatcher == nil:
				t.Fatalf("expected %#v got %#v", nil, err)
			case err == nil && tc.ErrorMatcher != nil:
				t.Fatalf("expected %#v got %#v", "error", nil)
			case !tc.ErrorMatcher(err):
				t.Fatalf("expected %#v got %#v", true, false)
			}

			if !reflect.DeepEqual(instanceToUpdate, tc.ExpectedInstanceToUpdate) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedInstanceToUpdate, instanceToUpdate)
			}
			if !reflect.DeepEqual(instanceToReimage, tc.ExpectedInstanceToReimage) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedInstanceToReimage, instanceToReimage)
			}
		})
	}
}

func Test_Resource_Instance_updateVersionParameterValue(t *testing.T) {
	testCases := []struct {
		Name                 string
		Instances            []compute.VirtualMachineScaleSetVM
		Instance             *compute.VirtualMachineScaleSetVM
		Version              string
		VersionValue         map[string]string
		ExpectedVersionValue map[string]string
		ErrorMatcher         func(err error) bool
	}{
		{
			Name:                 "case 0: empty input results in an empty JSON blob",
			Instances:            nil,
			Instance:             nil,
			Version:              "",
			VersionValue:         nil,
			ExpectedVersionValue: nil,
			ErrorMatcher:         nil,
		},
		{
			Name: "case 1: having an empty version bundle version blob and an instance and a version bundle version given results in a JSON blob with the instance ID and its version bundle version",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
				},
			},
			Instance:     nil,
			Version:      "0.1.0",
			VersionValue: nil,
			ExpectedVersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 2: having an empty version bundle version blob and instances and a version bundle version given results in a JSON blob with instance IDs and their version bundle version",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000003"),
				},
			},
			Instance:     nil,
			Version:      "0.1.0",
			VersionValue: nil,
			ExpectedVersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.1.0",
				"alq9y-worker-000003": "0.1.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 3: having a version bundle version blob and instances and a version bundle version given results in an updated JSON blob",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000003"),
				},
			},
			Instance: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("alq9y-worker-000001"),
			},
			Version: "0.2.0",
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.1.0",
				"alq9y-worker-000003": "0.1.0",
			},
			ExpectedVersionValue: map[string]string{
				"alq9y-worker-000001": "0.2.0",
				"alq9y-worker-000002": "0.1.0",
				"alq9y-worker-000003": "0.1.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 4: like 3 but with another instance and version",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000001"),
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
				},
				{
					InstanceID: to.StringPtr("alq9y-worker-000003"),
				},
			},
			Instance: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("alq9y-worker-000003"),
			},
			Version: "1.0.0",
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.1.0",
				"alq9y-worker-000003": "0.1.0",
			},
			ExpectedVersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.1.0",
				"alq9y-worker-000003": "1.0.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 5: instances being tracked in the version blob get removed when the instances do not exist anymore",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
				},
			},
			Instance: nil,
			Version:  "1.0.0",
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.1.0",
				"alq9y-worker-000003": "0.1.0",
			},
			ExpectedVersionValue: map[string]string{
				"alq9y-worker-000002": "0.1.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 6: instance removal and instance update works at the same time",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("alq9y-worker-000002"),
				},
			},
			Instance: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("alq9y-worker-000002"),
			},
			Version: "1.0.0",
			VersionValue: map[string]string{
				"alq9y-worker-000001": "0.1.0",
				"alq9y-worker-000002": "0.1.0",
				"alq9y-worker-000003": "0.1.0",
			},
			ExpectedVersionValue: map[string]string{
				"alq9y-worker-000002": "1.0.0",
			},
			ErrorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			versionValue, err := updateVersionParameterValue(tc.Instances, tc.Instance, tc.Version, tc.VersionValue)

			switch {
			case err == nil && tc.ErrorMatcher == nil:
				// fall through
			case err != nil && tc.ErrorMatcher == nil:
				t.Fatalf("expected %#v got %#v", nil, err)
			case err == nil && tc.ErrorMatcher != nil:
				t.Fatalf("expected %#v got %#v", "error", nil)
			case !tc.ErrorMatcher(err):
				t.Fatalf("expected %#v got %#v", true, false)
			}

			if !reflect.DeepEqual(versionValue, tc.ExpectedVersionValue) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedVersionValue, versionValue)
			}
		})
	}
}
