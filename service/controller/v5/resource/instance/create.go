package instance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v5/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v5/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	vmssDeploymentName = "cluster-vmss-template"
)

// EnsureCreated operates in 3 different stages which are executed sequentially.
// The first stage is for uploading ARM templates and is represented by stage
// DeploymentInitialized. The second stage is for waiting for ARM templates to
// be applied and is represented by stage ProvisioningSuccessful. the third
// stage is for draining and upgrading the VMSS instances and is represented by
// stage InstancesUpgrading. The stages are executed one after another and the
// instance resource cycles through them reliably until all necessary upgrade
// steps are successfully processed.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	if !resourceStatusExists(customObject, Stage) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

		computedDeployment, err := r.newDeployment(ctx, customObject, nil)
		if controllercontext.IsInvalidContext(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "missing dispatched output values in controller context")
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not ensure deployment")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			_, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(customObject), vmssDeploymentName, computedDeployment)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment")
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting resource status to '%s/%s'", Stage, DeploymentInitialized))

			err = r.setResourceStatus(customObject, Stage, DeploymentInitialized)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set resource status to '%s/%s'", Stage, DeploymentInitialized))
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}
	}

	if hasResourceStatus(customObject, Stage, DeploymentInitialized) {
		d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), vmssDeploymentName)
		if IsDeploymentNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "deployment not found")
			r.logger.LogCtx(ctx, "level", "debug", "message", "waiting for creation")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		s := *d.Properties.ProvisioningState
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

		if key.IsSucceededProvisioningState(s) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting resource status to '%s/%s'", Stage, ProvisioningSuccessful))

			err := r.setResourceStatus(customObject, Stage, ProvisioningSuccessful)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set resource status to '%s/%s'", Stage, ProvisioningSuccessful))
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		} else {
			r.debugger.LogFailedDeployment(ctx, d)

			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		}
	}

	if hasResourceStatus(customObject, Stage, ProvisioningSuccessful) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "vmss deployment successful")
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting resource status to '%s/%s'", Stage, InstancesUpgrading))

		err := r.setResourceStatus(customObject, Stage, InstancesUpgrading)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set resource status to '%s/%s'", Stage, InstancesUpgrading))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	if hasResourceStatus(customObject, Stage, InstancesUpgrading) {
		versionValue := map[string]string{}
		{
			for _, node := range customObject.Status.Cluster.Nodes {
				versionValue[node.Name] = node.Version
			}
		}

		var nodeConfigs []corev1alpha1.DrainerConfig
		{
			n := v1.NamespaceAll
			o := metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", key.ClusterIDLabel, key.ClusterID(customObject)),
			}

			list, err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).List(o)
			if err != nil {
				return microerror.Mask(err)
			}

			nodeConfigs = list.Items
		}

		var masterUpgraded bool
		{
			allMasterInstances, err := r.allInstances(ctx, customObject, key.MasterVMSSName)
			if IsScaleSetNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.MasterVMSSName(customObject)))
			} else if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", "processing master VMSSs")

				masterInstanceToUpdate, masterInstanceToDrain, masterInstanceToReimage, err := r.nextInstance(ctx, customObject, allMasterInstances, nodeConfigs, key.MasterInstanceName, versionValue)
				if err != nil {
					return microerror.Mask(err)
				}
				err = r.updateInstance(ctx, customObject, masterInstanceToUpdate, key.MasterVMSSName, key.MasterInstanceName)
				if err != nil {
					return microerror.Mask(err)
				}
				err = r.createDrainerConfig(ctx, customObject, masterInstanceToDrain, key.MasterInstanceName)
				if err != nil {
					return microerror.Mask(err)
				}
				err = r.reimageInstance(ctx, customObject, masterInstanceToReimage, key.MasterVMSSName, key.MasterInstanceName)
				if err != nil {
					return microerror.Mask(err)
				}
				err = r.deleteDrainerConfig(ctx, customObject, masterInstanceToReimage, key.MasterInstanceName, nodeConfigs)
				if err != nil {
					return microerror.Mask(err)
				}

				if masterInstanceToUpdate != nil || masterInstanceToDrain != nil || masterInstanceToReimage != nil {
					masterUpgraded = true
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", "processed master VMSSs")
			}
		}

		// In case the master instance is being updated we want to prevent any
		// other updates on the workers. This is because the update process
		// involves the draining of the updated node and if the master is being
		// updated at the same time the guest cluster's Kubernetes API is not
		// available in order to drain nodes. As consequence we have to reset the
		// worker instance selected to be reimaged in order to not update its
		// version information. The next reconciliation loop will catch up here
		// and instruct the worker instance to be reimaged again.
		var workerUpgraded bool
		if !masterUpgraded {
			allWorkerInstances, err := r.allInstances(ctx, customObject, key.WorkerVMSSName)
			if IsScaleSetNotFound(err) {
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(customObject)))
			} else if err != nil {
				return microerror.Mask(err)
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", "processing worker VMSSs")

				workerInstanceToUpdate, workerInstanceToDrain, workerInstanceToReimage, err := r.nextInstance(ctx, customObject, allWorkerInstances, nodeConfigs, key.WorkerInstanceName, versionValue)
				if err != nil {
					return microerror.Mask(err)
				}
				err = r.updateInstance(ctx, customObject, workerInstanceToUpdate, key.WorkerVMSSName, key.WorkerInstanceName)
				if err != nil {
					return microerror.Mask(err)
				}
				err = r.createDrainerConfig(ctx, customObject, workerInstanceToDrain, key.WorkerInstanceName)
				if err != nil {
					return microerror.Mask(err)
				}
				err = r.reimageInstance(ctx, customObject, workerInstanceToReimage, key.WorkerVMSSName, key.WorkerInstanceName)
				if err != nil {
					return microerror.Mask(err)
				}
				err = r.deleteDrainerConfig(ctx, customObject, workerInstanceToReimage, key.WorkerInstanceName, nodeConfigs)
				if err != nil {
					return microerror.Mask(err)
				}

				if workerInstanceToUpdate != nil || workerInstanceToDrain != nil || workerInstanceToReimage != nil {
					workerUpgraded = true
				}

				r.logger.LogCtx(ctx, "level", "debug", "message", "processed worker VMSSs")
			}
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "not processing worker VMSSs due to master VMSSs processing")
		}

		if !masterUpgraded && !workerUpgraded {
			r.logger.LogCtx(ctx, "level", "debug", "message", "neither masters nor workers upgraded")
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("removing resource status '%s/%s'", Stage, InstancesUpgrading))

			err := r.deleteResourceStatus(customObject, Stage, InstancesUpgrading)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("removed resource status '%s/%s'", Stage, InstancesUpgrading))
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		}
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

	var instances []compute.VirtualMachineScaleSetVM

	for result.NotDone() {
		instances = append(instances, result.Values()...)

		err := result.Next()
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found the scale set '%s'", deploymentNameFunc(customObject)))

	return instances, nil
}

func (r *Resource) createDrainerConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error)) error {
	if instance == nil {
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "creating node config for guest cluster node")

	instanceName, err := instanceNameFunc(customObject, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}

	n := key.ClusterID(customObject)
	c := &corev1alpha1.DrainerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				key.ClusterIDLabel: key.ClusterID(customObject),
			},
			Name: instanceName,
		},
		Spec: corev1alpha1.DrainerConfigSpec{
			Guest: corev1alpha1.DrainerConfigSpecGuest{
				Cluster: corev1alpha1.DrainerConfigSpecGuestCluster{
					API: corev1alpha1.DrainerConfigSpecGuestClusterAPI{
						Endpoint: key.ClusterAPIEndpoint(customObject),
					},
					ID: key.ClusterID(customObject),
				},
				Node: corev1alpha1.DrainerConfigSpecGuestNode{
					Name: instanceName,
				},
			},
			VersionBundle: corev1alpha1.DrainerConfigSpecVersionBundle{
				Version: "0.1.0",
			},
		},
	}

	_, err = r.g8sClient.CoreV1alpha1().DrainerConfigs(n).Create(c)
	if errors.IsAlreadyExists(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not create node config for guest cluster node")
		r.logger.LogCtx(ctx, "level", "debug", "message", "node config for guest cluster node does already exist")
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "created node config for guest cluster node")
	}

	return nil
}

func (r *Resource) deleteDrainerConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error), nodeConfigs []corev1alpha1.DrainerConfig) error {
	if instance == nil {
		return nil
	}

	instanceName, err := instanceNameFunc(customObject, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}

	if isNodeDrained(nodeConfigs, instanceName) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting node config for guest cluster node")

		var nodeConfigToRemove corev1alpha1.DrainerConfig
		for _, n := range nodeConfigs {
			if n.GetName() == instanceName {
				nodeConfigToRemove = n
				break
			}
		}

		n := nodeConfigToRemove.GetNamespace()
		i := nodeConfigToRemove.GetName()
		o := &metav1.DeleteOptions{}

		err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).Delete(i, o)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not delete node config for guest cluster node")
			r.logger.LogCtx(ctx, "level", "debug", "message", "node config for guest cluster node does not exist")
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "deleted node config for guest cluster node")
		}
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "not deleting node config for guest cluster node due to undrained node")
	}

	// TODO implement safety net to delete node configs that are over due for e.g. when node-operator fucks up

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
//     loop 4: worker 1 reimage
//     loop 5: worker 2 drained
//     loop 6: worker 2 reimage
//
func (r *Resource) nextInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM, nodeConfigs []corev1alpha1.DrainerConfig, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error), versionValue map[string]string) (*compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, error) {
	var err error

	var instanceToUpdate *compute.VirtualMachineScaleSetVM
	var instanceToDrain *compute.VirtualMachineScaleSetVM
	var instanceToReimage *compute.VirtualMachineScaleSetVM
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the next instance to be updated, drained or reimaged")

		instanceToUpdate, instanceToDrain, instanceToReimage, err = findActionableInstance(customObject, instances, nodeConfigs, instanceNameFunc, versionValue)
		if IsVersionBlobEmpty(err) {
			// When no version bundle version is found it means the cluster just got
			// created and the version bundle versions are not yet tracked within the
			// parameters of the guest cluster's VMSS deployment. In this case we must
			// not select an instance to be reimaged because we would roll a node that
			// just got created and is already up to date.
			r.logger.LogCtx(ctx, "level", "debug", "message", "no instance found to be updated, drained or reimaged")
			return nil, nil, nil, nil
		} else if err != nil {
			return nil, nil, nil, microerror.Mask(err)
		}

		if instanceToUpdate == nil && instanceToDrain == nil && instanceToReimage == nil {
			// Neither did we find an instance to be updated nor to be reimaged.
			// Nothing has to be done or we already processes all instances.
			r.logger.LogCtx(ctx, "level", "debug", "message", "no instance found to be updated, drained or reimaged")
			return nil, nil, nil, nil
		}

		if instanceToUpdate != nil {
			instanceName, err := instanceNameFunc(customObject, *instanceToUpdate.InstanceID)
			if err != nil {
				return nil, nil, nil, microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be updated", instanceName))
		}
		if instanceToDrain != nil {
			instanceName, err := instanceNameFunc(customObject, *instanceToDrain.InstanceID)
			if err != nil {
				return nil, nil, nil, microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be drained", instanceName))
		}
		if instanceToReimage != nil {
			instanceName, err := instanceNameFunc(customObject, *instanceToReimage.InstanceID)
			if err != nil {
				return nil, nil, nil, microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be reimaged", instanceName))
		}
	}

	return instanceToUpdate, instanceToDrain, instanceToReimage, nil
}

func (r *Resource) reimageInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error)) error {
	if instance == nil {
		return nil
	}

	instanceName, err := instanceNameFunc(customObject, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be reimaged", instanceName))

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

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be reimaged", instanceName))

	return nil
}

func (r *Resource) updateInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error)) error {
	if instance == nil {
		return nil
	}

	instanceName, err := instanceNameFunc(customObject, *instance.InstanceID)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be updated", instanceName))

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

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be updated", instanceName))

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
func findActionableInstance(customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM, nodeConfigs []corev1alpha1.DrainerConfig, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error), versionValue map[string]string) (*compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, *compute.VirtualMachineScaleSetVM, error) {
	var err error

	instanceInProgress := firstInstanceInProgress(customObject, instances)
	if instanceInProgress != nil {
		return nil, nil, nil, nil
	}

	var instanceToUpdate *compute.VirtualMachineScaleSetVM
	instanceToUpdate = firstInstanceToUpdate(customObject, instances)
	if instanceToUpdate != nil {
		return instanceToUpdate, nil, nil, nil
	}

	var instanceToReimage *compute.VirtualMachineScaleSetVM
	instanceToReimage, err = firstInstanceToReimage(customObject, instances, instanceNameFunc, versionValue)
	if err != nil {
		return nil, nil, nil, microerror.Mask(err)
	}
	if instanceToReimage != nil {
		instanceName, err := instanceNameFunc(customObject, *instanceToReimage.InstanceID)
		if err != nil {
			return nil, nil, nil, microerror.Mask(err)
		}
		if isNodeDrained(nodeConfigs, instanceName) {
			return nil, nil, instanceToReimage, nil
		} else {
			return nil, instanceToReimage, nil, nil
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
func firstInstanceToReimage(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error), versionValue map[string]string) (*compute.VirtualMachineScaleSetVM, error) {
	if versionValue == nil {
		return nil, microerror.Mask(versionBlobEmptyError)
	}

	for _, v := range list {
		desiredVersion := key.VersionBundleVersion(customObject)
		instanceName, err := instanceNameFunc(customObject, *v.InstanceID)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		instanceVersion, ok := versionValue[instanceName]
		if !ok {
			continue
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

func isNodeDrained(nodeConfigs []corev1alpha1.DrainerConfig, instanceName string) bool {
	for _, n := range nodeConfigs {
		if n.GetName() != instanceName {
			continue
		}
		if n.Status.HasDrainedCondition() {
			return true
		}
		if n.Status.HasTimeoutCondition() {
			return true
		}
	}

	return false
}
