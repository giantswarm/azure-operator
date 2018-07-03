package instance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/v3/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

const (
	masterVersionsKey  = "masterVersionBundleVersions"
	versionsKey        = "versionBundleVersions"
	vmssDeploymentName = "cluster-vmss-template"
	workerVersionsKey  = "workerVersionBundleVersions"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		cc, err := controllercontext.FromContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
		err = cc.Validate()
		if err != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", "missing dispatched output values in controller context")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")
			return nil
		}
	}

	{
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
		}
	}

	var masterVersionsValue map[string]string
	{
		d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), "master-vmss-deploy")
		if IsDeploymentNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			p, err := key.ToMap(d.Properties.Parameters)
			if err != nil {
				return microerror.Mask(err)
			}
			v, ok := p[versionsKey]
			if !ok {
				// fall through
			} else {
				m, err := key.ToMap(v)
				if err != nil {
					return microerror.Mask(err)
				}
				v, err := key.ToKeyValue(m)
				if err != nil {
					return microerror.Mask(err)
				}
				masterVersionsValue, err = key.ToStringMap(v)
				if err != nil {
					return microerror.Mask(err)
				}
			}
		}
	}

	var workerVersionsValue map[string]string
	{
		d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), "worker-vmss-deploy")
		if IsDeploymentNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			p, err := key.ToMap(d.Properties.Parameters)
			if err != nil {
				return microerror.Mask(err)
			}
			v, ok := p[versionsKey]
			if !ok {
				// fall through
			} else {
				m, err := key.ToMap(v)
				if err != nil {
					return microerror.Mask(err)
				}
				v, err := key.ToKeyValue(m)
				if err != nil {
					return microerror.Mask(err)
				}
				workerVersionsValue, err = key.ToStringMap(v)
				if err != nil {
					return microerror.Mask(err)
				}
			}
		}
	}

	var nodeConfigs []corev1alpha1.NodeConfig
	{
		n := v1.NamespaceAll
		o := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", key.ClusterIDLabel, key.ClusterID(customObject)),
		}

		list, err := r.g8sClient.CoreV1alpha1().NodeConfigs(n).List(o)
		if err != nil {
			return microerror.Mask(err)
		}

		nodeConfigs = list.Items
	}

	var allMasterInstances []compute.VirtualMachineScaleSetVM
	var drainedMasterInstance *compute.VirtualMachineScaleSetVM
	var reimagedMasterInstance *compute.VirtualMachineScaleSetVM
	var updatedMasterInstance *compute.VirtualMachineScaleSetVM
	{
		allMasterInstances, err = r.allInstances(ctx, customObject, key.MasterVMSSName)
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.MasterVMSSName(customObject)))
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "processing master VMSSs")

			updatedMasterInstance, drainedMasterInstance, reimagedMasterInstance, err = r.nextInstance(ctx, customObject, allMasterInstances, nodeConfigs, key.MasterInstanceName, masterVersionsValue)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.updateInstance(ctx, customObject, updatedMasterInstance, key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.createNodeConfig(ctx, customObject, drainedMasterInstance, key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.reimageInstance(ctx, customObject, reimagedMasterInstance, key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.deleteNodeConfig(ctx, customObject, reimagedMasterInstance, key.MasterInstanceName, nodeConfigs)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "processed master VMSSs")
		}
	}

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	var drainedWorkerInstance *compute.VirtualMachineScaleSetVM
	var reimagedWorkerInstance *compute.VirtualMachineScaleSetVM
	var updatedWorkerInstance *compute.VirtualMachineScaleSetVM
	{
		allWorkerInstances, err = r.allInstances(ctx, customObject, key.WorkerVMSSName)
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(customObject)))
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "processing worker VMSSs")

			updatedWorkerInstance, drainedWorkerInstance, reimagedWorkerInstance, err = r.nextInstance(ctx, customObject, allWorkerInstances, nodeConfigs, key.WorkerInstanceName, workerVersionsValue)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.updateInstance(ctx, customObject, updatedWorkerInstance, key.WorkerVMSSName, key.WorkerInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.createNodeConfig(ctx, customObject, drainedWorkerInstance, key.WorkerInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			// In case the master instance is being updated we want to prevent any
			// other updates on the workers. This is because the update process
			// involves the draining of the updated node and if the master is being
			// updated at the same time the guest cluster's Kubernetes API is not
			// available in order to drain nodes. As consequence we have to reset the
			// worker instance selected to be reimaged in order to not update its
			// version information. The next reconciliation loop will catch up here
			// and instruct the worker instance to be reimaged again.
			if reimagedMasterInstance == nil {
				err = r.reimageInstance(ctx, customObject, reimagedWorkerInstance, key.WorkerVMSSName, key.WorkerInstanceName)
				if err != nil {
					return microerror.Mask(err)
				}
				err = r.deleteNodeConfig(ctx, customObject, reimagedWorkerInstance, key.WorkerInstanceName, nodeConfigs)
				if err != nil {
					return microerror.Mask(err)
				}
			} else if reimagedWorkerInstance != nil {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("not ensuring instance '%s' to be reimaged due to master processing", key.WorkerInstanceName(customObject, *reimagedWorkerInstance.InstanceID)))
				reimagedWorkerInstance = nil
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "processed worker VMSSs")
		}
	}

	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

		masterVersionsValue, err := newVersionParameterValue(allMasterInstances, reimagedMasterInstance, key.VersionBundleVersion(customObject), masterVersionsValue)
		if err != nil {
			return microerror.Mask(err)
		}
		workerVersionsValue, err := newVersionParameterValue(allWorkerInstances, reimagedWorkerInstance, key.VersionBundleVersion(customObject), workerVersionsValue)
		if err != nil {
			return microerror.Mask(err)
		}
		params := map[string]interface{}{
			masterVersionsKey: masterVersionsValue,
			workerVersionsKey: workerVersionsValue,
		}
		computedDeployment, err := r.newDeployment(ctx, customObject, params)
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

	c, err := r.getVMsClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := deploymentNameFunc(customObject)
	result, err := c.List(ctx, g, s, "", "", "")
	if IsScaleSetNotFound(err) {
		return nil, microerror.Mask(scaleSetNotFoundError)
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found the scale set '%s'", deploymentNameFunc(customObject)))

	return result.Values(), nil
}

func (r *Resource) createNodeConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "creating node config for guest cluster node")

	n := customObject.GetNamespace()
	c := &corev1alpha1.NodeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				key.ClusterIDLabel: key.ClusterID(customObject),
			},
			Name: instanceNameFunc(customObject, *instance.InstanceID),
		},
		Spec: corev1alpha1.NodeConfigSpec{
			Guest: corev1alpha1.NodeConfigSpecGuest{
				Cluster: corev1alpha1.NodeConfigSpecGuestCluster{
					API: corev1alpha1.NodeConfigSpecGuestClusterAPI{
						Endpoint: key.ClusterAPIEndpoint(customObject),
					},
					ID: key.ClusterID(customObject),
				},
				Node: corev1alpha1.NodeConfigSpecGuestNode{
					Name: instanceNameFunc(customObject, *instance.InstanceID),
				},
			},
			VersionBundle: corev1alpha1.NodeConfigSpecVersionBundle{
				Version: "0.1.0",
			},
		},
	}

	_, err := r.g8sClient.CoreV1alpha1().NodeConfigs(n).Create(c)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "created node config for guest cluster node")

	return nil
}

// nextInstance finds the next instance to either be updated, drained or
// reimaged. There always only be one of either options at the same time. We
// only either update an instance, drain an instance, or reimage it. The order
// of actions across multiple reconciliation loops is to update all instances
// first, then drain them, then reimage them. Each step of the three different
// processes is being executed in its own reconciliation loop. The mechanism is
// applied to all of the available instances until they got into the desired
// state.
//
//     loop 1: worker 1 update
//     loop 2: worker 2 update
//     loop 3: worker 1 drained
//     loop 4: worker 2 drained
//     loop 5: worker 1 reimage
//     loop 6: worker 2 reimage
//
func (r *Resource) nextInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM, nodeConfigs []corev1alpha1.NodeConfig, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string, versionValue map[string]string) (*compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, error) {
	var err error

	var instanceToUpdate *compute.VirtualMachineScaleSetVM
	var instanceToDrain *compute.VirtualMachineScaleSetVM
	var instanceToReimage *compute.VirtualMachineScaleSetVM
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the next instance to be updated or reimaged")

		instanceToUpdate, instanceToDrain, instanceToReimage, err = findActionableInstance(customObject, instances, nodeConfigs, instanceNameFunc, versionValue)
		if IsVersionBlobEmpty(err) {
			// When no version bundle version is found it means the cluster just got
			// created and the version bundle versions are not yet tracked within the
			// parameters of the guest cluster's VMSS deployment. In this case we must
			// not select an instance to be reimaged because we would roll a node that
			// just got created and is already up to date.
			r.logger.LogCtx(ctx, "level", "debug", "message", "version blob still empty")
			return nil, nil, nil, nil
		} else if err != nil {
			return nil, nil, nil, microerror.Mask(err)
		}

		if instanceToUpdate == nil && instanceToDrain == nil && instanceToReimage == nil {
			// Neither did we find an instance to be updated nor to be reimaged.
			// Nothing has to be done or we already processes all instances.
			r.logger.LogCtx(ctx, "level", "debug", "message", "no instance found to be updated or reimaged")
			return nil, nil, nil, nil
		}

		if instanceToUpdate != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be updated", instanceNameFunc(customObject, *instanceToUpdate.InstanceID)))
		}
		if instanceToDrain != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be drained", instanceNameFunc(customObject, *instanceToDrain.InstanceID)))
		}
		if instanceToReimage != nil {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be reimaged", instanceNameFunc(customObject, *instanceToReimage.InstanceID)))
		}
	}

	return instanceToUpdate, instanceToReimage, instanceToDrain, nil
}

func (r *Resource) reimageInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be reimaged", instanceNameFunc(customObject, *instance.InstanceID)))

	c, err := r.getScaleSetsClient(ctx)
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

func (r *Resource) deleteNodeConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string, nodeConfigs []corev1alpha1.NodeConfig) error {
	if instance == nil {
		return nil
	}

	if isNodeDrained(nodeConfigs, instanceNameFunc(customObject, *instance.InstanceID)) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting node config for guest cluster node")

		var nodeConfigToRemove corev1alpha1.NodeConfig
		for _, n := range nodeConfigs {
			if n.GetName() == instanceNameFunc(customObject, *instance.InstanceID) {
				nodeConfigToRemove = n
				break
			}
		}

		n := nodeConfigToRemove.GetNamespace()
		i := nodeConfigToRemove.GetName()
		o := &metav1.DeleteOptions{}

		err := r.g8sClient.CoreV1alpha1().NodeConfigs(n).Delete(i, o)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "deleted node config for guest cluster node")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "not deleting node config for guest cluster node due to undrained node")
	}

	// TODO implement safety net to delete node configs that are over due for e.g. when node-operator fucks up

	return nil
}

func (r *Resource) updateInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be updated", instanceNameFunc(customObject, *instance.InstanceID)))

	c, err := r.getScaleSetsClient(ctx)
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

func containsInstanceID(list []compute.VirtualMachineScaleSetVM, id string) bool {
	for _, v := range list {
		if *v.InstanceID == id {
			return true
		}
	}

	return false
}

// findActionableInstance either returns an instance to update or an instance to
// reimage, but never both at the same time.
func findActionableInstance(customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM, nodeConfigs []corev1alpha1.NodeConfig, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string, versionValue map[string]string) (*compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, error) {
	var err error

	instanceInProgress := firstInstanceInProgress(customObject, instances)
	if instanceInProgress != nil {
		return nil, nil, nil, nil
	}

	var instanceToUpdate *compute.VirtualMachineScaleSetVM
	if instanceInProgress == nil {
		instanceToUpdate = firstInstanceToUpdate(customObject, instances)
		if instanceToUpdate != nil {
			return instanceToUpdate, nil, nil, nil
		}
	}

	var instanceToReimage *compute.VirtualMachineScaleSetVM
	if instanceToUpdate == nil {
		instanceToReimage, err = firstInstanceToReimage(customObject, instances, versionValue)
		if err != nil {
			return nil, nil, nil, microerror.Mask(err)
		}
		if instanceToReimage != nil {
			if isNodeDrained(nodeConfigs, instanceNameFunc(customObject, *instanceToReimage.InstanceID)) {
				return nil, nil, instanceToReimage, nil
			} else {
				return nil, instanceToReimage, nil, nil
			}
		}
	}

	return nil, nil, nil, nil
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
func firstInstanceToReimage(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM, versionValue map[string]string) (*compute.VirtualMachineScaleSetVM, error) {
	if versionValue == nil {
		return nil, microerror.Mask(versionBlobEmptyError)
	}

	for _, v := range list {
		desiredVersion := key.VersionBundleVersion(customObject)
		instanceVersion, ok := versionValue[*v.InstanceID]
		if !ok {
			return nil, microerror.Mask(versionBlobEmptyError)
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

func isNodeDrained(nodeConfigs []corev1alpha1.NodeConfig, instanceName string) bool {
	for _, n := range nodeConfigs {
		if n.GetName() == instanceName && n.Status.HasFinalCondition() {
			return true
		}
	}

	return false
}

func newVersionParameterValue(list []compute.VirtualMachineScaleSetVM, reimagedInstance *compute.VirtualMachineScaleSetVM, version string, versionValue map[string]string) (map[string]string, error) {
	// ignore empty
	if len(list) == 0 && versionValue == nil {
		return map[string]string{}, nil
	}

	// fill empty
	if len(list) != 0 && len(versionValue) == 0 {
		m := map[string]string{}
		for _, v := range list {
			m[*v.InstanceID] = version
		}

		return m, nil
	}

	// remove missing
	if len(versionValue) != 0 {
		m := map[string]string{}
		for k, v := range versionValue {
			if !containsInstanceID(list, k) {
				continue
			}
			m[k] = v
		}

		versionValue = m
	}

	// update existing
	if len(versionValue) != 0 {
		if reimagedInstance != nil {
			versionValue[*reimagedInstance.InstanceID] = version
		}

		return versionValue, nil
	}

	return nil, microerror.Mask(invalidConfigError)
}
