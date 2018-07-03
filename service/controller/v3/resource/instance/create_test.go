package instance

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Resource_Instance_findActionableInstance(t *testing.T) {
	testCases := []struct {
		Name                      string
		CustomObject              providerv1alpha1.AzureConfig
		Instances                 []compute.VirtualMachineScaleSetVM
		NodeConfigs               []corev1alpha1.NodeConfig
		InstanceNameFunc          func(customObject providerv1alpha1.AzureConfig, instanceID string) string
		VersionValue              map[string]string
		ExpectedInstanceToUpdate  *compute.VirtualMachineScaleSetVM
		ExpectedInstanceToDrain   *compute.VirtualMachineScaleSetVM
		ExpectedInstanceToReimage *compute.VirtualMachineScaleSetVM
		ErrorMatcher              func(err error) bool
	}{
		{
			Name:                      "case 0: empty input results in no action",
			CustomObject:              providerv1alpha1.AzureConfig{},
			Instances:                 nil,
			NodeConfigs:               nil,
			InstanceNameFunc:          key.WorkerInstanceName,
			VersionValue:              nil,
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              IsVersionBlobEmpty,
		},
		{
			Name: "case 1: one instance being up to date results in no action",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newFinalNodeConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.1.0",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 2: two instances being up to date results in no action",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newFinalNodeConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.1.0",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 3: one instance not having the latest model applied results in updating the instance",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newFinalNodeConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.1.0",
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 4: two instances not having the latest model applied results in updating the first instance",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newFinalNodeConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.1.0",
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 5: one instance having the latest model applied and one instance not having the latest model applied results in updating the second instance",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newFinalNodeConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.1.0",
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("2"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 6: two instances not having the latest version bundle version applied results in reimaging the first instance",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newFinalNodeConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.0.1",
				"2": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain:  nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
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
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newFinalNodeConfigForID("al9qy-worker-000002"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain:  nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("2"),
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
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newFinalNodeConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.0.1",
				"2": "0.0.1",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 9: one instance having the latest version bundle version applied and one instance not having the latest version bundle version applied and being in provisioning state 'InProgress' results in no action",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newFinalNodeConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.0.1",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 10: instances that should be reimaged should be drained first",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newIdleNodeConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.0.1",
				"2": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 11: same as 10 but with different input",
			CustomObject: providerv1alpha1.AzureConfig{
				Spec: providerv1alpha1.AzureConfigSpec{
					Cluster: providerv1alpha1.Cluster{
						ID: "al9qy",
					},
					VersionBundle: providerv1alpha1.AzureConfigSpecVersionBundle{
						Version: "0.1.0",
					},
				},
			},
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
					},
				},
			},
			NodeConfigs: []corev1alpha1.NodeConfig{
				newIdleNodeConfigForID("al9qy-worker-000002"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("2"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
				},
			},
			ExpectedInstanceToReimage: nil,
			ErrorMatcher:              nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			instanceToUpdate, instanceToDrain, instanceToReimage, err := findActionableInstance(tc.CustomObject, tc.Instances, tc.NodeConfigs, tc.InstanceNameFunc, tc.VersionValue)

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
			if !reflect.DeepEqual(instanceToDrain, tc.ExpectedInstanceToDrain) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedInstanceToDrain, instanceToDrain)
			}
			if !reflect.DeepEqual(instanceToReimage, tc.ExpectedInstanceToReimage) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedInstanceToReimage, instanceToReimage)
			}
		})
	}
}

func Test_Resource_Instance_newVersionParameterValue(t *testing.T) {
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
			ExpectedVersionValue: map[string]string{},
			ErrorMatcher:         nil,
		},
		{
			Name: "case 1: having an empty version bundle version blob and an instance and a version bundle version given results in a JSON blob with the instance ID and its version bundle version",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
				},
			},
			Instance:     nil,
			Version:      "0.1.0",
			VersionValue: map[string]string{},
			ExpectedVersionValue: map[string]string{
				"1": "0.1.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 2: having an empty version bundle version blob and instances and a version bundle version given results in a JSON blob with instance IDs and their version bundle version",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
				},
				{
					InstanceID: to.StringPtr("2"),
				},
				{
					InstanceID: to.StringPtr("3"),
				},
			},
			Instance:     nil,
			Version:      "0.1.0",
			VersionValue: map[string]string{},
			ExpectedVersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.1.0",
				"3": "0.1.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 3: having a version bundle version blob and instances and a version bundle version given results in an updated JSON blob",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
				},
				{
					InstanceID: to.StringPtr("2"),
				},
				{
					InstanceID: to.StringPtr("3"),
				},
			},
			Instance: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
			},
			Version: "0.2.0",
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.1.0",
				"3": "0.1.0",
			},
			ExpectedVersionValue: map[string]string{
				"1": "0.2.0",
				"2": "0.1.0",
				"3": "0.1.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 4: like 3 but with another instance and version",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("1"),
				},
				{
					InstanceID: to.StringPtr("2"),
				},
				{
					InstanceID: to.StringPtr("3"),
				},
			},
			Instance: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("3"),
			},
			Version: "1.0.0",
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.1.0",
				"3": "0.1.0",
			},
			ExpectedVersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.1.0",
				"3": "1.0.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 5: instances being tracked in the version blob get removed when the instances do not exist anymore",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("2"),
				},
			},
			Instance: nil,
			Version:  "1.0.0",
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.1.0",
				"3": "0.1.0",
			},
			ExpectedVersionValue: map[string]string{
				"2": "0.1.0",
			},
			ErrorMatcher: nil,
		},
		{
			Name: "case 6: instance removal and instance update works at the same time",
			Instances: []compute.VirtualMachineScaleSetVM{
				{
					InstanceID: to.StringPtr("2"),
				},
			},
			Instance: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("2"),
			},
			Version: "1.0.0",
			VersionValue: map[string]string{
				"1": "0.1.0",
				"2": "0.1.0",
				"3": "0.1.0",
			},
			ExpectedVersionValue: map[string]string{
				"2": "1.0.0",
			},
			ErrorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			versionValue, err := newVersionParameterValue(tc.Instances, tc.Instance, tc.Version, tc.VersionValue)

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

func newFinalNodeConfigForID(id string) corev1alpha1.NodeConfig {
	return corev1alpha1.NodeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Status: corev1alpha1.NodeConfigStatus{
			Conditions: []corev1alpha1.NodeConfigStatusCondition{
				{
					Status: corev1alpha1.NodeConfigStatusStatusTrue,
					Type:   corev1alpha1.NodeConfigStatusTypeDrained,
				},
			},
		},
	}
}

func newIdleNodeConfigForID(id string) corev1alpha1.NodeConfig {
	return corev1alpha1.NodeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Status: corev1alpha1.NodeConfigStatus{
			Conditions: []corev1alpha1.NodeConfigStatusCondition{},
		},
	}
}
