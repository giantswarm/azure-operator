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

	"github.com/giantswarm/azure-operator/service/controller/v3/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

const (
	masterBlobKey      = "masterVersionBundleVersions"
	vmssDeploymentName = "cluster-vmss-template"
	workerBlobKey      = "workerVersionBundleVersions"
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
			parameters, err = key.ToMap(fetchedDeployment.Properties.Parameters)
			if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	var allMasterInstances []compute.VirtualMachineScaleSetVM
	var reimagedMasterInstance *compute.VirtualMachineScaleSetVM
	var updatedMasterInstance *compute.VirtualMachineScaleSetVM
	{
		allMasterInstances, err = r.allInstances(ctx, customObject, key.MasterVMSSName)
		if IsScaleSetNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "processing master VMSSs")

			updatedMasterInstance, reimagedMasterInstance, err = r.nextInstance(ctx, customObject, allMasterInstances, key.MasterInstanceName, parameters[masterBlobKey])
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.updateInstance(ctx, customObject, updatedMasterInstance, key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.reimageInstance(ctx, customObject, reimagedMasterInstance, key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "processed master VMSSs")
		}
	}

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	var reimagedWorkerInstance *compute.VirtualMachineScaleSetVM
	var updatedWorkerInstance *compute.VirtualMachineScaleSetVM
	{
		allWorkerInstances, err = r.allInstances(ctx, customObject, key.WorkerVMSSName)
		if IsScaleSetNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "processing worker VMSSs")

			updatedWorkerInstance, reimagedWorkerInstance, err = r.nextInstance(ctx, customObject, allWorkerInstances, key.WorkerInstanceName, parameters[workerBlobKey])
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.updateInstance(ctx, customObject, updatedWorkerInstance, key.WorkerVMSSName, key.WorkerInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			if reimagedMasterInstance == nil {
				// In case the master instance is being updated we want to prevent any
				// other updates on the workers. This is because the update process
				// involves the draining of the updated node and if the master is being
				// updated at the same time the guest cluster's Kubernetes API is not
				// available in order to drain nodes.
				err = r.reimageInstance(ctx, customObject, reimagedWorkerInstance, key.WorkerVMSSName, key.WorkerInstanceName)
				if err != nil {
					return microerror.Mask(err)
				}
			} else if reimagedWorkerInstance != nil {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("not ensuring instance '%s' to be reimaged due to master processing", key.WorkerInstanceName(customObject, *reimagedWorkerInstance.InstanceID)))
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "processed worker VMSSs")
		}
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

		masterBlobValue, err := updateVersionParameterValue(allMasterInstances, reimagedMasterInstance, key.VersionBundleVersion(customObject), parameters[masterBlobKey])
		if err != nil {
			return microerror.Mask(err)
		}
		fmt.Printf("4: %#v\n", masterBlobValue)
		workerBlobValue, err := updateVersionParameterValue(allWorkerInstances, reimagedWorkerInstance, key.VersionBundleVersion(customObject), parameters[workerBlobKey])
		if err != nil {
			return microerror.Mask(err)
		}
		fmt.Printf("5: %#v\n", workerBlobValue)
		params := map[string]interface{}{
			masterBlobKey: masterBlobValue,
			workerBlobKey: workerBlobValue,
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
		if IsVersionBlobEmpty(err) {
			// When no version bundle version is found it means the cluster just got
			// created and the version bundle versions are not yet tracked within the
			// parameters of the guest cluster's VMSS deployment. In this case we must
			// not select an instance to be reimaged because we would roll a node that
			// just got created and is already up to date.
			r.logger.LogCtx(ctx, "level", "debug", "message", "version blob still empty")
			return nil, nil, nil
		} else if err != nil {
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

func containsInstanceVersion(list []compute.VirtualMachineScaleSetVM, version string) bool {
	for _, v := range list {
		if *v.InstanceID == version {
			return true
		}
	}

	return false
}

// findActionableInstance either returns an instance to update or an instance to
// reimage, but never both at the same time.
func findActionableInstance(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM, value interface{}) (*compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, error) {
	var err error

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
		instanceToReimage, err = firstInstanceToReimage(customObject, list, value)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}
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
func firstInstanceToReimage(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM, value interface{}) (*compute.VirtualMachineScaleSetVM, error) {
	for _, v := range list {
		desiredVersion := key.VersionBundleVersion(customObject)
		instanceVersion, err := versionBundleVersionForInstance(&v, value)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		if desiredVersion == instanceVersion {
			continue
		}

		return &v, nil
	}

	return nil, nil
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

func updateVersionParameterValue(list []compute.VirtualMachineScaleSetVM, reimagedInstance *compute.VirtualMachineScaleSetVM, version string, value interface{}) (string, error) {
	// init empty
	if len(list) == 0 && value == nil {
		fmt.Printf("1\n")
		return "{}", nil
	}

	// parse existing
	var versionMap map[string]string
	if len(list) != 0 && value != nil {
		fmt.Printf("3\n")
		m, err := key.ToMap(value)
		if err != nil {
			return "", microerror.Mask(err)
		}
		s, err := key.ToKeyValue(m)
		if err != nil {
			return "", microerror.Mask(err)
		}

		err = json.Unmarshal([]byte(s), &versionMap)
		if err != nil {
			return "", microerror.Mask(err)
		}
	}

	// fill empty
	if len(versionMap) == 0 {
		fmt.Printf("2\n")
		m := map[string]string{}
		for _, v := range list {
			m[*v.InstanceID] = version
		}

		b, err := json.Marshal(m)
		if err != nil {
			return "", microerror.Mask(err)
		}

		return string(b), nil
	}

	// remove missing
	if len(versionMap) != 0 {
		m := map[string]string{}
		for k, v := range versionMap {
			if !containsInstanceVersion(list, k) {
				continue
			}
			m[k] = v
		}

		versionMap = m
	}

	// update existing
	if len(versionMap) != 0 {
		if reimagedInstance != nil {
			versionMap[*reimagedInstance.InstanceID] = version
		}

		b, err := json.Marshal(versionMap)
		if err != nil {
			return "", microerror.Mask(err)
		}

		return string(b), nil
	}

	return "", microerror.Mask(invalidConfigError)
}

func versionBundleVersionForInstance(instance *compute.VirtualMachineScaleSetVM, value interface{}) (string, error) {
	if value == nil {
		return "", microerror.Mask(versionBlobEmptyError)
	}

	m, err := key.ToMap(value)
	if err != nil {
		return "", microerror.Mask(err)
	}
	s, err := key.ToKeyValue(m)
	if err != nil {
		return "", microerror.Mask(err)
	}

	var d map[string]string
	err = json.Unmarshal([]byte(s), &d)
	if err != nil {
		return "", microerror.Mask(err)
	}

	version, ok := d[*instance.InstanceID]
	if !ok {
		return "", microerror.Mask(versionBlobEmptyError)
	}

	return version, nil
}
