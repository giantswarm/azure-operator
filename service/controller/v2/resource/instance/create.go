package instance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var allMasterInstances []compute.VirtualMachineScaleSetVM
	var masterInstance *compute.VirtualMachineScaleSetVM
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "processing master VMSSs")

		allMasterInstances, masterInstance, err = r.processInstance(ctx, customObject, key.MasterVMSSName, key.MasterInstanceName)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "processed master VMSSs")
	}

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	var workerInstance *compute.VirtualMachineScaleSetVM
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "processing worker VMSSs")

		allWorkerInstances, workerInstance, err = r.processInstance(ctx, customObject, key.WorkerVMSSName, key.WorkerInstanceName)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "processed worker VMSSs")
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

		var deployment azureresource.Deployment

		d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), mainDeploymentName)
		if IsNotFound(err) {
			params := map[string]interface{}{
				"masterVersionBundleVersions": "{}",
				"workerVersionBundleVersions": "{}",
			}
			deployment, err = r.newDeployment(customObject, params)
			if err != nil {
				return microerror.Mask(err)
			}
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			s := *d.Properties.ProvisioningState
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

			if !key.IsFinalProvisioningState(s) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

				return nil
			}

			params := map[string]interface{}{
				"masterVersionBundleVersions": TODO,
				"workerVersionBundleVersions": TODO,
			}
			deployment, err = r.newDeployment(customObject, params)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		_, err = deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(customObject), mainDeploymentName, deployment)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment")
	}

	return nil
}

func (r *Resource) findInstances(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string) ([]compute.VirtualMachineScaleSetVM, error) {
	c, err := r.getVMsClient()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := deploymentNameFunc(customObject)
	result, err := c.List(ctx, g, s, "", "", "")
	if IsScaleSetNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the scale set")

		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return result.Values(), nil
}

// nextInstance finds the next instance to either be updated or reimaged. There
// must never be both of either options at the same time. We only either update
// an instance or reimage it. The order of actions across multiple
// reconciliation loops is to update all instances first and then reimage them.
// Each step of the two different processes is being executed in its own
// reconciliation loop. The mechanism is applied to all of the available
// instances until they got into the desired state.
//
//     loop 1: worker 1 update
//     loop 2: worker 2 update
//     loop 3: worker 3 update
//     loop 4: worker 1 reimage
//     loop 5: worker 2 reimage
//     loop 6: worker 3 reimage
//
func (r *Resource) nextInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM) (*compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the next instance to be updated or reimaged")

	instanceToUpdate, instanceToReimage, err := findActionableInstance(customObject, instances)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	if instanceToUpdate == nil && instanceToReimage == nil {
		// Neither did we find an instance to be updated nor to be reimaged.
		// Nothing has to be done or we already processes all instances.
		r.logger.LogCtx(ctx, "level", "debug", "message", "no instance found to be updated or reimaged")

		return nil, nil, nil
	}

	if instanceToUpdate != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be updated", instanceNameFunc(customObject, *instanceToUpdate.InstanceID)))
	}
	if instanceToReimage != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be reimaged", instanceNameFunc(customObject, *instanceToReimage.InstanceID)))
	}

	return instanceToUpdate, instanceToReimage, nil
}

func (r *Resource) processInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	// Trigger the update for the found instance.
	if instanceToUpdate != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be updated", instanceNameFunc(customObject, *instanceToUpdate.InstanceID)))

		c, err := r.getScaleSetsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		g := key.ResourceGroupName(customObject)
		s := deploymentNameFunc(customObject)
		ids := compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
			InstanceIds: to.StringSlicePtr([]string{
				*instanceToUpdate.InstanceID,
			}),
		}
		_, err = c.UpdateInstances(ctx, g, s, ids)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be updated", instanceNameFunc(customObject, *instanceToUpdate.InstanceID)))
	}

	// Trigger the reimage for the found instance.
	if instanceToReimage != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be reimaged", instanceNameFunc(customObject, *instanceToReimage.InstanceID)))

		c, err := r.getScaleSetsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		g := key.ResourceGroupName(customObject)
		s := deploymentNameFunc(customObject)
		ids := &compute.VirtualMachineScaleSetVMInstanceIDs{
			InstanceIds: to.StringSlicePtr([]string{
				*instanceToReimage.InstanceID,
			}),
		}
		_, err = c.Reimage(ctx, g, s, ids)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be reimaged", instanceNameFunc(customObject, *instanceToReimage.InstanceID)))
	}

	return nil
}

// findActionableInstance either returns an instance to update or an instance to
// reimage, but never both at the same time.
func findActionableInstance(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM) (*compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, error) {
	for _, i := range list {
		if i.ProvisioningState == nil {
			continue
		}
		fmt.Printf("%#v\n", *i.InstanceID)
		fmt.Printf("%#v\n", i.Tags)
		for k, v := range i.Tags {
			fmt.Printf("%#v\n", k)
			fmt.Printf("%#v\n", v)
		}
	}
	instanceInProgress := firstInstanceInProgress(customObject, list)

	fmt.Printf("1\n")

	var instanceToUpdate *compute.VirtualMachineScaleSetVM
	if instanceInProgress == nil {
		fmt.Printf("2\n")
		instanceToUpdate = firstInstanceToUpdate(customObject, list)
	}

	var instanceToReimage *compute.VirtualMachineScaleSetVM
	if instanceToUpdate == nil {
		fmt.Printf("3\n")
		instanceToReimage = firstInstanceToReimage(customObject, list)
	}

	return instanceToUpdate, instanceToReimage, nil
}

// firstInstanceInProgress returns the first instance in the list not having a
// final state. In case all instances are in a final state
// firstInstanceInProgress returns nil.
func firstInstanceInProgress(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM) *compute.VirtualMachineScaleSetVM {
	for _, v := range list {
		if v.ProvisioningState == nil || key.IsFinalProvisioningState(*v.ProvisioningState) {
			continue
		}

		return &v
	}

	return nil
}

// firstInstanceToReimage returns the first instance to be reimaged. The
// decision of reimaging an instance is done by comparing the desired version
// bundle version of the custom object and the current version bundle version of
// the instance's tags applied. In case all instances are reimaged
// firstInstanceToReimage return nil.
func firstInstanceToReimage(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM) *compute.VirtualMachineScaleSetVM {
	for _, v := range list {
		if key.VersionBundleVersion(customObject) == versionBundleVersionForInstance(v) {
			continue
		}

		return &v
	}

	return nil
}

// firstInstanceToUpdate return the first instance to be updated. The decision
// of updating an instance is done by checking if the latest scale set model is
// applied. In case all instances are updated firstInstanceToUpdate return nil.
func firstInstanceToUpdate(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM) *compute.VirtualMachineScaleSetVM {
	for _, v := range list {
		if *v.LatestModelApplied {
			continue
		}

		return &v
	}

	return nil
}

func setVersionBundleVersion(instance compute.VirtualMachineScaleSetVM, version string) compute.VirtualMachineScaleSetVM {
	blob, ok := instance.Tags["versionBundleVersions"]
	if !ok {
		panic("missing tags")
	}

	var m map[string]string
	err := json.Unmarshal([]byte(blob), &m)
	if err != nil {
		panic(err)
	}

	m[*v.InstanceID] = version

	raw, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	instance.Tags["versionBundleVersions"] = string(raw)

	return instance
}

func versionBundleVersionForInstance(v compute.VirtualMachineScaleSetVM) string {
	blob, ok := v.Tags["versionBundleVersions"]
	if !ok {
		panic("missing tags")
	}

	var m map[string]string
	err := json.Unmarshal([]byte(blob), &m)
	if err != nil {
		panic(err)
	}

	version, ok := m[*v.InstanceID]
	if !ok {
		panic("missing id")
	}

	return version
}
