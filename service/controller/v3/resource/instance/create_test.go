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
		ExpectedInstanceToUpdate  *compute.VirtualMachineScaleSetVM
		ExpectedInstanceToReimage *compute.VirtualMachineScaleSetVM
		ErrorMatcher              func(err error) bool
	}{
		{
			Name:                      "case 0: empty input results in no action",
			CustomObject:              providerv1alpha1.AzureConfig{},
			Instances:                 []compute.VirtualMachineScaleSetVM{},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
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
					ID: to.StringPtr("alq9y-worker-000001"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.1.0"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
					},
				},
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
					ID: to.StringPtr("alq9y-worker-000001"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.1.0"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
					},
				},
				{
					ID: to.StringPtr("alq9y-worker-000002"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.1.0"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
					},
				},
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
					ID: to.StringPtr("alq9y-worker-000001"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.1.0"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
					},
				},
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				ID: to.StringPtr("alq9y-worker-000001"),
				Tags: map[string]*string{
					"versionBundleVersion": to.StringPtr("0.1.0"),
				},
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
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
					ID: to.StringPtr("alq9y-worker-000001"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.1.0"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
					},
				},
				{
					ID: to.StringPtr("alq9y-worker-000002"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.1.0"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
					},
				},
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				ID: to.StringPtr("alq9y-worker-000001"),
				Tags: map[string]*string{
					"versionBundleVersion": to.StringPtr("0.1.0"),
				},
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
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
					ID: to.StringPtr("alq9y-worker-000001"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.1.0"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
					},
				},
				{
					ID: to.StringPtr("alq9y-worker-000002"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.1.0"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
					},
				},
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				ID: to.StringPtr("alq9y-worker-000002"),
				Tags: map[string]*string{
					"versionBundleVersion": to.StringPtr("0.1.0"),
				},
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
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
					ID: to.StringPtr("alq9y-worker-000001"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.0.1"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
					},
				},
				{
					ID: to.StringPtr("alq9y-worker-000002"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.0.1"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
					},
				},
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				ID: to.StringPtr("alq9y-worker-000001"),
				Tags: map[string]*string{
					"versionBundleVersion": to.StringPtr("0.0.1"),
				},
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
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
					ID: to.StringPtr("alq9y-worker-000001"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.1.0"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
					},
				},
				{
					ID: to.StringPtr("alq9y-worker-000002"),
					Tags: map[string]*string{
						"versionBundleVersion": to.StringPtr("0.0.1"),
					},
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
					},
				},
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				ID: to.StringPtr("alq9y-worker-000002"),
				Tags: map[string]*string{
					"versionBundleVersion": to.StringPtr("0.0.1"),
				},
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
				},
			},
			ErrorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			instanceToUpdate, instanceToReimage, err := findActionableInstance(tc.CustomObject, tc.Instances)

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
