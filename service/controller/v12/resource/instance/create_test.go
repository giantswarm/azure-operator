package instance

import (
	"github.com/Azure/go-autorest/autorest/to"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Resource_Instance_findActionableInstance(t *testing.T) {
	const dockerDiskName = "DockerDisk"
	const kubeletDiskName = "KubeletDisk"
	oneGBDataDisks := []compute.DataDisk{
		compute.DataDisk{
			Name:       to.StringPtr(dockerDiskName),
			DiskSizeGB: to.Int32Ptr(1),
		},
		compute.DataDisk{
			Name:       to.StringPtr(kubeletDiskName),
			DiskSizeGB: to.Int32Ptr(1),
		},
	}

	desiredDiskSizeOneGb := map[string]int32{
		dockerDiskName:  int32(1),
		kubeletDiskName: int32(1),
	}

	desiredDiskSizesDockerTwoGb := map[string]int32{
		dockerDiskName:  int32(2),
		kubeletDiskName: int32(1),
	}

	desiredDiskSizesKubeletTwoGb := map[string]int32{
		dockerDiskName:  int32(1),
		kubeletDiskName: int32(2),
	}

	testCases := []struct {
		Name                      string
		CustomObject              providerv1alpha1.AzureConfig
		Instances                 []compute.VirtualMachineScaleSetVM
		DrainerConfigs            []corev1alpha1.DrainerConfig
		InstanceNameFunc          func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error)
		VersionValue              map[string]string
		ExpectedInstanceToUpdate  *compute.VirtualMachineScaleSetVM
		ExpectedInstanceToDrain   *compute.VirtualMachineScaleSetVM
		ExpectedInstanceToReimage *compute.VirtualMachineScaleSetVM
		DesiredDiskSizes          map[string]int32
		ErrorMatcher              func(err error) bool
	}{
		{
			Name:                      "case 0: empty input results in no action",
			CustomObject:              providerv1alpha1.AzureConfig{},
			Instances:                 nil,
			DrainerConfigs:            nil,
			InstanceNameFunc:          key.WorkerInstanceName,
			VersionValue:              nil,
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
				"al9qy-worker-000002": "0.1.0",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
					ProvisioningState:  to.StringPtr("Succeeded"),
					StorageProfile: &compute.StorageProfile{
						DataDisks: &oneGBDataDisks,
					},
				},
			},
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
				"al9qy-worker-000002": "0.1.0",
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
					ProvisioningState:  to.StringPtr("Succeeded"),
					StorageProfile: &compute.StorageProfile{
						DataDisks: &oneGBDataDisks,
					},
				},
			},
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(false),
						ProvisioningState:  to.StringPtr("Succeeded"),
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
				"al9qy-worker-000002": "0.1.0",
			},
			ExpectedInstanceToUpdate: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("2"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(false),
					ProvisioningState:  to.StringPtr("Succeeded"),
					StorageProfile: &compute.StorageProfile{
						DataDisks: &oneGBDataDisks,
					},
				},
			},
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.0.1",
				"al9qy-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain:  nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
					StorageProfile: &compute.StorageProfile{
						DataDisks: &oneGBDataDisks,
					},
				},
			},
			DesiredDiskSizes: desiredDiskSizeOneGb,
			ErrorMatcher:     nil,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000002"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
				"al9qy-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain:  nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("2"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
					StorageProfile: &compute.StorageProfile{
						DataDisks: &oneGBDataDisks,
					},
				},
			},
			DesiredDiskSizes: desiredDiskSizeOneGb,
			ErrorMatcher:     nil,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.0.1",
				"al9qy-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("InProgress"),
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
				"al9qy-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate:  nil,
			ExpectedInstanceToDrain:   nil,
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newIdleDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.0.1",
				"al9qy-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
					StorageProfile: &compute.StorageProfile{
						DataDisks: &oneGBDataDisks,
					},
				},
			},
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
				{
					InstanceID: to.StringPtr("2"),
					VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
						LatestModelApplied: to.BoolPtr(true),
						ProvisioningState:  to.StringPtr("Succeeded"),
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newIdleDrainerConfigForID("al9qy-worker-000002"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
				"al9qy-worker-000002": "0.0.1",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("2"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
					StorageProfile: &compute.StorageProfile{
						DataDisks: &oneGBDataDisks,
					},
				},
			},
			ExpectedInstanceToReimage: nil,
			DesiredDiskSizes:          desiredDiskSizeOneGb,
			ErrorMatcher:              nil,
		},
		{
			Name: "case 12: one instance being up to date, docker disk size changed",
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain:  nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
					StorageProfile: &compute.StorageProfile{
						DataDisks: &oneGBDataDisks,
					},
				},
			},
			DesiredDiskSizes: desiredDiskSizesDockerTwoGb,
			ErrorMatcher:     nil,
		},
		{
			Name: "case 13: one instance being up to date, kubelet disk size changed",
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
						StorageProfile: &compute.StorageProfile{
							DataDisks: &oneGBDataDisks,
						},
					},
				},
			},
			DrainerConfigs: []corev1alpha1.DrainerConfig{
				newFinalDrainerConfigForID("al9qy-worker-000001"),
			},
			InstanceNameFunc: key.WorkerInstanceName,
			VersionValue: map[string]string{
				"al9qy-worker-000001": "0.1.0",
			},
			ExpectedInstanceToUpdate: nil,
			ExpectedInstanceToDrain:  nil,
			ExpectedInstanceToReimage: &compute.VirtualMachineScaleSetVM{
				InstanceID: to.StringPtr("1"),
				VirtualMachineScaleSetVMProperties: &compute.VirtualMachineScaleSetVMProperties{
					LatestModelApplied: to.BoolPtr(true),
					ProvisioningState:  to.StringPtr("Succeeded"),
					StorageProfile: &compute.StorageProfile{
						DataDisks: &oneGBDataDisks,
					},
				},
			},
			DesiredDiskSizes: desiredDiskSizesKubeletTwoGb,
			ErrorMatcher:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ws, err := getWorkingSet(tc.CustomObject, tc.Instances, tc.DrainerConfigs, tc.InstanceNameFunc, tc.VersionValue)

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

			if !reflect.DeepEqual(ws.InstanceToUpdate(), tc.ExpectedInstanceToUpdate) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedInstanceToUpdate, ws.instanceToUpdate)
			}
			if !reflect.DeepEqual(ws.InstanceToDrain(), tc.ExpectedInstanceToDrain) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedInstanceToDrain, ws.instanceToDrain)
			}
			if !reflect.DeepEqual(ws.InstanceToReimage(), tc.ExpectedInstanceToReimage) {
				t.Fatalf("expected %#v got %#v", tc.ExpectedInstanceToReimage, ws.instanceToReimage)
			}
		})
	}
}

func newFinalDrainerConfigForID(id string) corev1alpha1.DrainerConfig {
	return corev1alpha1.DrainerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Status: corev1alpha1.DrainerConfigStatus{
			Conditions: []corev1alpha1.DrainerConfigStatusCondition{
				{
					Status: corev1alpha1.DrainerConfigStatusStatusTrue,
					Type:   corev1alpha1.DrainerConfigStatusTypeDrained,
				},
			},
		},
	}
}

func newIdleDrainerConfigForID(id string) corev1alpha1.DrainerConfig {
	return corev1alpha1.DrainerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Status: corev1alpha1.DrainerConfigStatus{
			Conditions: []corev1alpha1.DrainerConfigStatusCondition{},
		},
	}
}
