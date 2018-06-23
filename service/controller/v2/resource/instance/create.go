package instance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v2/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

const (
	vmssDeploymentName = "cluster-vmss-template"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var fetchedDeployment *azureresource.DeploymentExtended
	var parameters map[string]interface{}
	{
		deploymentsClient, err := r.getDeploymentsClient()
		if err != nil {
			return microerror.Mask(err)
		}
		d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), vmssDeploymentName)
		if IsDeploymentNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			s := *d.Properties.ProvisioningState
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

			if !key.IsFinalProvisioningState(s) {
				r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

				return nil
			}

			fetchedDeployment = &d
			// TODO error handling
			parameters = fetchedDeployment.Properties.Parameters.(map[string]interface{})
		}
	}

	var allMasterInstances []compute.VirtualMachineScaleSetVM
	var updatedMasterInstance *compute.VirtualMachineScaleSetVM
	{
		allMasterInstances, err = r.allInstances(ctx, customObject, key.MasterVMSSName)
		if IsScaleSetNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "processing master VMSSs")

			instanceToUpdate, instanceToReimage, err := r.nextInstance(ctx, customObject, allMasterInstances, key.MasterInstanceName, parameters["masterVersionBundleVersions"])
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.updateInstance(ctx, customObject, instanceToUpdate, key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.reimageInstance(ctx, customObject, instanceToReimage, key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			updatedMasterInstance = instanceToReimage

			r.logger.LogCtx(ctx, "level", "debug", "message", "processed master VMSSs")
		}
	}

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	var updatedWorkerInstance *compute.VirtualMachineScaleSetVM
	{
		allWorkerInstances, err = r.allInstances(ctx, customObject, key.WorkerVMSSName)
		if IsScaleSetNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "processing worker VMSSs")

			instanceToUpdate, instanceToReimage, err := r.nextInstance(ctx, customObject, allWorkerInstances, key.WorkerInstanceName, parameters["workerVersionBundleVersions"])
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.updateInstance(ctx, customObject, instanceToUpdate, key.WorkerVMSSName, key.WorkerInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.reimageInstance(ctx, customObject, instanceToReimage, key.WorkerVMSSName, key.WorkerInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			updatedWorkerInstance = instanceToReimage

			r.logger.LogCtx(ctx, "level", "debug", "message", "processed worker VMSSs")
		}
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

		var masterBlob string
		var workerBlob string
		if fetchedDeployment == nil {
			masterBlob = "{}"
			workerBlob = "{}"
		} else {
			masterBlob = updateVersionParameterValue(allMasterInstances, updatedMasterInstance, key.VersionBundleVersion(customObject), parameters["masterVersionBundleVersions"])
			workerBlob = updateVersionParameterValue(allWorkerInstances, updatedWorkerInstance, key.VersionBundleVersion(customObject), parameters["workerVersionBundleVersions"])
		}

		params := map[string]interface{}{
			"masterVersionBundleVersions": masterBlob,
			"workerVersionBundleVersions": workerBlob,
		}
		computedDeployment, err := r.newDeployment(ctx, customObject, params)
		if controllercontext.IsInvalidContext(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "missing dispatched output values in controller context")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		deploymentsClient, err := r.getDeploymentsClient()
		if err != nil {
			return microerror.Mask(err)
		}
		_, err = deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(customObject), vmssDeploymentName, computedDeployment)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment")
	}

	return nil
}

func (r *Resource) allInstances(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string) ([]compute.VirtualMachineScaleSetVM, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for the scale set '%s'", deploymentNameFunc(customObject)))

	c, err := r.getVMsClient()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := deploymentNameFunc(customObject)
	result, err := c.List(ctx, g, s, "", "", "")
	if IsScaleSetNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", deploymentNameFunc(customObject)))

		return nil, microerror.Mask(scaleSetNotFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found the scale set '%s'", deploymentNameFunc(customObject)))

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
func (r *Resource) nextInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string, value interface{}) (*compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, error) {
	var err error

	var instanceToUpdate *compute.VirtualMachineScaleSetVM
	var instanceToReimage *compute.VirtualMachineScaleSetVM
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the next instance to be updated or reimaged")

		instanceToUpdate, instanceToReimage, err = findActionableInstance(customObject, instances, value)
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
	}

	return instanceToUpdate, instanceToReimage, nil
}

func (r *Resource) reimageInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be reimaged", instanceNameFunc(customObject, *instance.InstanceID)))

	c, err := r.getScaleSetsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := deploymentNameFunc(customObject)
	ids := &compute.VirtualMachineScaleSetVMInstanceIDs{
		InstanceIds: to.StringSlicePtr([]string{
			*instance.InstanceID,
		}),
	}
	_, err = c.Reimage(ctx, g, s, ids)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be reimaged", instanceNameFunc(customObject, *instance.InstanceID)))

	return nil
}

func (r *Resource) updateInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be updated", instanceNameFunc(customObject, *instance.InstanceID)))

	c, err := r.getScaleSetsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := deploymentNameFunc(customObject)
	ids := compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
		InstanceIds: to.StringSlicePtr([]string{
			*instance.InstanceID,
		}),
	}
	_, err = c.UpdateInstances(ctx, g, s, ids)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be updated", instanceNameFunc(customObject, *instance.InstanceID)))

	return nil
}

// findActionableInstance either returns an instance to update or an instance to
// reimage, but never both at the same time.
func findActionableInstance(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM, value interface{}) (*compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, error) {
	instanceInProgress := firstInstanceInProgress(customObject, list)
	if instanceInProgress != nil {
		return nil, nil, nil
	}

	var instanceToUpdate *compute.VirtualMachineScaleSetVM
	if instanceInProgress == nil {
		instanceToUpdate = firstInstanceToUpdate(customObject, list)
	}

	var instanceToReimage *compute.VirtualMachineScaleSetVM
	if instanceToUpdate == nil {
		instanceToReimage = firstInstanceToReimage(customObject, list, value)
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
func firstInstanceToReimage(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM, value interface{}) *compute.VirtualMachineScaleSetVM {
	for _, v := range list {
		if key.VersionBundleVersion(customObject) == versionBundleVersionForInstance(&v, value) {
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

func updateVersionParameterValue(list []compute.VirtualMachineScaleSetVM, instance *compute.VirtualMachineScaleSetVM, version string, value interface{}) string {
	fmt.Printf("value: %#v\n", value)
	fmt.Printf("version: %#v\n", version)

	var blob string
	{
		m, ok := value.(map[string]interface{})
		if !ok {
			//		return "", microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, v)
			// TODO error handling
			return ""
		}
		v, ok := m["value"]
		if !ok {
			//		return "", microerror.Maskf(missingParameterValueError, "value")
			// TODO error handling
			return ""
		}
		// TODO error handling
		blob = v.(string)
		fmt.Printf("blob: %#v\n", blob)
	}

	if blob == "{}" {
		m := map[string]string{}
		for _, v := range list {
			m[*v.InstanceID] = version
		}

		b, err := json.Marshal(m)
		if err != nil {
			// TODO error handling
			return ""
		}
		fmt.Printf("b: %s\n", b)

		return string(b)
	}

	// In case the given instance is nil there is nothing to change and we just
	// return what we got.
	if instance == nil {
		// TODO error handling
		return blob
	}

	// Here we got an instance which implies we have to update its version bundle
	// version carried in the paramter value.
	var raw string
	{
		m, ok := value.(map[string]interface{})
		if !ok {
			//		return "", microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, v)
			// TODO error handling
			return ""
		}

		m[*instance.InstanceID] = version

		b, err := json.Marshal(m)
		if err != nil {
			// TODO error handling
			return ""
		}

		raw = string(b)
	}

	return raw
}

func versionBundleVersionForInstance(instance *compute.VirtualMachineScaleSetVM, value interface{}) string {
	m, ok := value.(map[string]interface{})
	if !ok {
		//		return "", microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, v)
		// TODO error handling
		return ""
	}
	v, ok := m["value"]
	if !ok {
		//		return "", microerror.Maskf(missingParameterValueError, "value")
		// TODO error handling
		return ""
	}
	var d map[string]string
	err := json.Unmarshal([]byte(v.(string)), &d)
	if err != nil {
		// TODO error handling
		return ""
	}
	version, ok := d[*instance.InstanceID]
	if !ok {
		// instance is not yet tracked
		return ""
	}

	return version
}
