package instance

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v12/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/v12/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v12/key"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if !resourceStatusExists(customObject, Stage) {
		return r.handleResourceStatusMissing(ctx, customObject)
	}

	if hasResourceStatus(customObject, Stage, DeploymentInitialized) {
		return r.handleDeploymentInitializedStatus(ctx, customObject)
	}

	if hasResourceStatus(customObject, Stage, DeploymentCompleted) {
		return r.handleDeploymentCompletedStatus(ctx, customObject)
	}

	if hasResourceStatus(customObject, Stage, ProvisioningSuccessful) {
		return r.handleProvisioningSuccessfulStatus(ctx, customObject)
	}

	if hasResourceStatus(customObject, Stage, InstancesUpgrading) {
		return r.handleInstancesUpgradingStatus(ctx, customObject)
	}

	return nil
}

func (r *Resource) handleResourceStatusMissing(ctx context.Context, customObject providerv1alpha1.AzureConfig) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	computedDeployment, err := r.newDeployment(ctx, customObject, nil)
	if controllercontext.IsInvalidContext(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "missing dispatched output values in controller context")
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not ensure deployment")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	} else if blobclient.IsBlobNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		res, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(customObject), key.VmssDeploymentName, computedDeployment)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = deploymentsClient.CreateOrUpdateResponder(res.Response())
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

		deploymentTemplateChk, err := getDeploymentTemplateChecksum(computedDeployment)
		if err != nil {
			return microerror.Mask(err)
		}

		if deploymentTemplateChk != "" {
			err = r.setResourceStatus(customObject, DeploymentTemplateChecksum, deploymentTemplateChk)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentTemplateChecksum, deploymentTemplateChk))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentTemplateChecksum))
			// todo remove any DeploymentTemplateChecksum leftovers from the CR
		}

		deploymentParametersChk, err := getDeploymentParametersChecksum(computedDeployment)
		if err != nil {
			return microerror.Mask(err)
		}

		if deploymentTemplateChk != "" {
			err = r.setResourceStatus(customObject, DeploymentParametersChecksum, deploymentParametersChk)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentParametersChecksum, deploymentParametersChk))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentParametersChecksum))
			// todo remove any DeploymentParametersChecksum leftovers from the CR
		}

		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}
}

func (r *Resource) handleDeploymentInitializedStatus(ctx context.Context, customObject providerv1alpha1.AzureConfig) error {
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), key.VmssDeploymentName)
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
		r.debugger.LogFailedDeployment(ctx, d, err)

		if key.IsFinalProvisioningState(s) {
			// Deployment is not running and not succeeded (Failed?)
			// This indicates some kind of error in the deployment template and/or parameters.
			// Deleting the resource status will force the next loop to apply the deployment once again.
			// (If the azure operator has been fixed/updated in the meantime that could lead to a fix).
			err := r.deleteResourceStatus(customObject, Stage, DeploymentInitialized)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("removed resource status '%s/%s'", Stage, DeploymentInitialized))
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}
}

func (r *Resource) handleDeploymentCompletedStatus(ctx context.Context, customObject providerv1alpha1.AzureConfig) error {
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), key.VmssDeploymentName)
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
		computedDeployment, err := r.newDeployment(ctx, customObject, nil)
		if controllercontext.IsInvalidContext(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "missing dispatched output values in controller context")
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not ensure deployment")
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		} else if blobclient.IsBlobNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
			resourcecanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			desiredDeploymentTemplateChk, err := getDeploymentTemplateChecksum(computedDeployment)
			if err != nil {
				return microerror.Mask(err)
			}

			desiredDeploymentParametersChk, err := getDeploymentParametersChecksum(computedDeployment)
			if err != nil {
				return microerror.Mask(err)
			}

			currentDeploymentTemplateChk, err := r.getResourceStatus(customObject, DeploymentTemplateChecksum)
			if err != nil {
				return microerror.Mask(err)
			}

			currentDeploymentParametersChk, err := r.getResourceStatus(customObject, DeploymentParametersChecksum)
			if err != nil {
				return microerror.Mask(err)
			}

			if currentDeploymentTemplateChk != desiredDeploymentTemplateChk || currentDeploymentParametersChk != desiredDeploymentParametersChk {
				r.logger.LogCtx(ctx, "level", "debug", "message", "Either the template or parameters are changed")

				err := r.deleteResourceStatus(customObject, Stage, DeploymentCompleted)
				if err != nil {
					return microerror.Mask(err)
				}
				r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("removed resource status '%s/%s'", Stage, DeploymentCompleted))
			} else {
				r.logger.LogCtx(ctx, "level", "debug", "message", "Template and parameters unchanged")
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}
	} else if key.IsFinalProvisioningState(s) {
		// deployment is failed
		err := r.deleteResourceStatus(customObject, Stage, DeploymentCompleted)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("removed resource status '%s/%s'", Stage, DeploymentCompleted))
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	// If the flow arrives here, the deployment is running and we have to wait for it to complete.
	return nil
}

func (r *Resource) handleProvisioningSuccessfulStatus(ctx context.Context, customObject providerv1alpha1.AzureConfig) error {
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

func (r *Resource) handleInstancesUpgradingStatus(ctx context.Context, customObject providerv1alpha1.AzureConfig) error {
	versionValue := map[string]string{}
	{
		for _, node := range customObject.Status.Cluster.Nodes {
			versionValue[node.Name] = node.Version
		}
	}

	var drainerConfigs []corev1alpha1.DrainerConfig
	{
		n := v1.NamespaceAll
		o := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", key.ClusterIDLabel, key.ClusterID(customObject)),
		}

		list, err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).List(o)
		if err != nil {
			return microerror.Mask(err)
		}

		drainerConfigs = list.Items
	}

	var masterUpgradeInProgress bool
	{
		allMasterInstances, err := r.allInstances(ctx, customObject, key.MasterVMSSName)
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.MasterVMSSName(customObject)))
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "processing master VMSSs")

			ws, err := r.nextInstance(ctx, customObject, allMasterInstances, drainerConfigs, key.MasterInstanceName, versionValue)
			if err != nil {
				return microerror.Mask(err)
			}

			err = r.updateInstance(ctx, customObject, ws.InstanceToUpdate(), key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.createDrainerConfig(ctx, customObject, ws.InstanceToDrain(), key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.reimageInstance(ctx, customObject, ws.InstanceToReimage(), key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.deleteDrainerConfig(ctx, customObject, ws.InstanceToReimage(), key.MasterInstanceName, drainerConfigs)
			if err != nil {
				return microerror.Mask(err)
			}

			masterUpgradeInProgress = ws.IsWIP()
			r.logger.LogCtx(ctx, "level", "debug", "message", "processed master VMSSs")
		}
	}

	// In case the master instance is being updated we want to prevent any
	// other updates on the workers. This is because the update process
	// involves the draining of the updated node and if the master is being
	// updated at the same time the tenant cluster's Kubernetes API is not
	// available in order to drain nodes. As consequence we have to reset the
	// worker instance selected to be reimaged in order to not update its
	// version information. The next reconciliation loop will catch up here
	// and instruct the worker instance to be reimaged again.
	var workerUpgradeInProgess bool
	if !masterUpgradeInProgress {
		allWorkerInstances, err := r.allInstances(ctx, customObject, key.WorkerVMSSName)
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.WorkerVMSSName(customObject)))
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "processing worker VMSSs")

			ws, err := r.nextInstance(ctx, customObject, allWorkerInstances, drainerConfigs, key.WorkerInstanceName, versionValue)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.updateInstance(ctx, customObject, ws.InstanceToUpdate(), key.WorkerVMSSName, key.WorkerInstanceName)

			err = r.updateInstance(ctx, customObject, ws.instanceToUpdate, key.WorkerVMSSName, key.WorkerInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.createDrainerConfig(ctx, customObject, ws.InstanceToDrain(), key.WorkerInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.reimageInstance(ctx, customObject, ws.InstanceToReimage(), key.WorkerVMSSName, key.WorkerInstanceName)
			if err != nil {
				return microerror.Mask(err)
			}
			err = r.deleteDrainerConfig(ctx, customObject, ws.InstanceToReimage(), key.WorkerInstanceName, drainerConfigs)
			if err != nil {
				return microerror.Mask(err)
			}

			workerUpgradeInProgess = ws.IsWIP()
			r.logger.LogCtx(ctx, "level", "debug", "message", "processed worker VMSSs")
		}
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "not processing worker VMSSs due to master VMSSs processing")
	}

	if !masterUpgradeInProgress && !workerUpgradeInProgess {
		r.logger.LogCtx(ctx, "level", "debug", "message", "neither masters nor workers upgraded")

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("setting resource status to '%s/%s'", Stage, DeploymentCompleted))

		err := r.setResourceStatus(customObject, Stage, DeploymentCompleted)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set resource status to '%s/%s'", Stage, DeploymentCompleted))

		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
	}

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
func (r *Resource) nextInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM, drainerConfigs []corev1alpha1.DrainerConfig, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) (string, error), versionValue map[string]string) (*workingSet, error) {
	var err error

	var ws *workingSet
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the next instance to be updated, drained or reimaged")

		ws, err = getWorkingSet(customObject, instances, drainerConfigs, instanceNameFunc, versionValue)
		if IsVersionBlobEmpty(err) {
			// When no version bundle version is found it means the cluster just got
			// created and the version bundle versions are not yet tracked within the
			// parameters of the tenant cluster's VMSS deployment. In this case we
			// must not select an instance to be reimaged because we would roll a node
			// that just got created and is already up to date.
			r.logger.LogCtx(ctx, "level", "debug", "message", "no instance found to be updated, drained or reimaged")
			return nil, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		if !ws.IsWIP() {
			// Neither did we find an instance to be updated nor to be reimaged.
			// Nothing has to be done or we already processes all instances.
			r.logger.LogCtx(ctx, "level", "debug", "message", "no instance found to be updated, drained or reimaged")
			return nil, nil
		}

		if ws.InstanceToUpdate() != nil {
			instanceName, err := instanceNameFunc(customObject, *ws.InstanceToUpdate().InstanceID)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be updated", instanceName))
		}
		if ws.InstanceToDrain() != nil {
			instanceName, err := instanceNameFunc(customObject, *ws.InstanceToDrain().InstanceID)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be drained", instanceName))
		}
		if ws.InstanceToReimage() != nil {
			instanceName, err := instanceNameFunc(customObject, *ws.InstanceToReimage().InstanceID)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be reimaged", instanceName))
		}
	}

	return ws, nil
}
