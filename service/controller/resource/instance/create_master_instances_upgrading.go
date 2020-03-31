package instance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/resource/instance/internal/state"
)

func (r *Resource) masterInstancesUpgradingTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	versionValue := map[string]string{}
	{
		for _, node := range cr.Status.Cluster.Nodes {
			versionValue[node.Name] = node.Version
		}
	}

	var drainerConfigs []corev1alpha1.DrainerConfig
	{
		n := v1.NamespaceAll
		o := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", key.ClusterIDLabel, key.ClusterID(cr)),
		}

		list, err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).List(o)
		if err != nil {
			return "", microerror.Mask(err)
		}

		drainerConfigs = list.Items
	}

	var masterUpgradeInProgress bool
	{
		allMasterInstances, err := r.allInstances(ctx, cr, key.MasterVMSSName)
		if IsScaleSetNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.MasterVMSSName(cr))) // nolint: errcheck
		} else if err != nil {
			return "", microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "processing master VMSSs") // nolint: errcheck

			ws, err := r.nextInstance(ctx, cr, allMasterInstances, drainerConfigs, key.MasterInstanceName, versionValue)
			if err != nil {
				return "", microerror.Mask(err)
			}

			err = r.updateInstance(ctx, cr, ws.InstanceToUpdate(), key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return "", microerror.Mask(err)
			}
			if ws.InstanceToDrain() != nil {
				err = r.createDrainerConfig(ctx, cr, key.MasterInstanceName(cr, *ws.InstanceToDrain().InstanceID))
				if err != nil {
					return "", microerror.Mask(err)
				}
			}
			err = r.reimageInstance(ctx, cr, ws.InstanceToReimage(), key.MasterVMSSName, key.MasterInstanceName)
			if err != nil {
				return "", microerror.Mask(err)
			}
			err = r.deleteDrainerConfig(ctx, cr, ws.InstanceToReimage(), key.MasterInstanceName, drainerConfigs)
			if err != nil {
				return "", microerror.Mask(err)
			}

			masterUpgradeInProgress = ws.IsWIP()

			r.logger.LogCtx(ctx, "level", "debug", "message", "processed master VMSSs") // nolint: errcheck
		}
	}

	if !masterUpgradeInProgress {
		// When masters are upgraded, proceed to workers.
		return WaitForMastersToBecomeReady, nil
	}

	// Upgrade still in progress. Keep current state.
	return currentState, nil
}

func (r *Resource) allInstances(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string) ([]compute.VirtualMachineScaleSetVM, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("looking for the scale set '%s'", deploymentNameFunc(customObject))) // nolint: errcheck

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

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found the scale set '%s'", deploymentNameFunc(customObject))) // nolint: errcheck

	return instances, nil
}

func (r *Resource) createDrainerConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, nodeName string) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "creating drainer config for tenant cluster node") // nolint: errcheck

	n := key.ClusterID(customObject)
	c := &corev1alpha1.DrainerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				key.ClusterIDLabel: key.ClusterID(customObject),
			},
			Name: nodeName,
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
					Name: nodeName,
				},
			},
			VersionBundle: corev1alpha1.DrainerConfigSpecVersionBundle{
				Version: "0.2.0",
			},
		},
	}

	_, err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).Create(c)
	if errors.IsAlreadyExists(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not create drainer config for tenant cluster node")     // nolint: errcheck
		r.logger.LogCtx(ctx, "level", "debug", "message", "drainer config for tenant cluster node does already exist") // nolint: errcheck
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "created drainer config for tenant cluster node") // nolint: errcheck
	}

	return nil
}

func (r *Resource) deleteDrainerConfig(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string, drainerConfigs []corev1alpha1.DrainerConfig) error {
	if instance == nil {
		return nil
	}

	instanceName := instanceNameFunc(customObject, *instance.InstanceID)

	if isNodeDrained(drainerConfigs, instanceName) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting drainer config for tenant cluster node") // nolint: errcheck

		var drainerConfigToRemove corev1alpha1.DrainerConfig
		for _, n := range drainerConfigs {
			if n.GetName() == instanceName {
				drainerConfigToRemove = n
				break
			}
		}

		n := drainerConfigToRemove.GetNamespace()
		i := drainerConfigToRemove.GetName()
		o := &metav1.DeleteOptions{}

		err := r.g8sClient.CoreV1alpha1().DrainerConfigs(n).Delete(i, o)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not delete drainer config for tenant cluster node") // nolint: errcheck
			r.logger.LogCtx(ctx, "level", "debug", "message", "drainer config for tenant cluster node does not exist") // nolint: errcheck
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "deleted drainer config for tenant cluster node") // nolint: errcheck
		}
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "not deleting drainer config for tenant cluster node due to undrained node") // nolint: errcheck
	}

	// TODO implement safety net to delete drainer configs that are over due for e.g. when node-operator fucks up

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
func (r *Resource) nextInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM, drainerConfigs []corev1alpha1.DrainerConfig, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string, versionValue map[string]string) (*workingSet, error) {
	var err error

	var ws *workingSet
	{
		r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the next instance to be updated, drained or reimaged") // nolint: errcheck

		ws, err = getWorkingSet(customObject, instances, drainerConfigs, instanceNameFunc, versionValue)
		if IsVersionBlobEmpty(err) {
			// When no version bundle version is found it means the cluster just got
			// created and the version bundle versions are not yet tracked within the
			// parameters of the tenant cluster's VMSS deployment. In this case we
			// must not select an instance to be reimaged because we would roll a node
			// that just got created and is already up to date.
			r.logger.LogCtx(ctx, "level", "debug", "message", "no instance found to be updated, drained or reimaged") // nolint: errcheck
			return nil, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		if !ws.IsWIP() {
			// Neither did we find an instance to be updated nor to be reimaged.
			// Nothing has to be done or we already processes all instances.
			r.logger.LogCtx(ctx, "level", "debug", "message", "no instance found to be updated, drained or reimaged") // nolint: errcheck
			return nil, nil
		}

		if ws.InstanceToUpdate() != nil {
			instanceName := instanceNameFunc(customObject, *ws.InstanceToUpdate().InstanceID)
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be updated", instanceName)) // nolint: errcheck
		}
		if ws.InstanceToDrain() != nil {
			instanceName := instanceNameFunc(customObject, *ws.InstanceToDrain().InstanceID)
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be drained", instanceName)) // nolint: errcheck
		}
		if ws.InstanceToReimage() != nil {
			instanceName := instanceNameFunc(customObject, *ws.InstanceToReimage().InstanceID)
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found instance '%s' has to be reimaged", instanceName)) // nolint: errcheck
		}
	}

	return ws, nil
}

func (r *Resource) reimageInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	instanceName := instanceNameFunc(customObject, *instance.InstanceID)

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be reimaged", instanceName)) // nolint: errcheck

	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	g := key.ResourceGroupName(customObject)
	s := deploymentNameFunc(customObject)
	ids := &compute.VirtualMachineScaleSetReimageParameters{
		InstanceIds: to.StringSlicePtr([]string{
			*instance.InstanceID,
		}),
	}
	res, err := c.Reimage(ctx, g, s, ids)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = c.ReimageResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	go func() {
		err := r.startInstanceWatchdog(ctx, g, s)
		if err != nil {
			r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("Watchdog failed for instance '%s'", instanceName)) // nolint: errcheck
		}
	}()

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be reimaged", instanceName)) // nolint: errcheck

	return nil
}

func (r *Resource) updateInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	instanceName := instanceNameFunc(customObject, *instance.InstanceID)

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be updated", instanceName)) // nolint: errcheck

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
	res, err := c.UpdateInstances(ctx, g, s, ids)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = c.UpdateInstancesResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be updated", instanceName)) // nolint: errcheck

	go func() {
		err := r.startInstanceWatchdog(ctx, g, s)
		if err != nil {
			r.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("Watchdog failed for instance '%s'", instanceName)) // nolint: errcheck
		}
	}()

	return nil
}

// getWorkingSet either returns an instance to update or an instance to
// reimage, but never both at the same time.
func getWorkingSet(customObject providerv1alpha1.AzureConfig, instances []compute.VirtualMachineScaleSetVM, drainerConfigs []corev1alpha1.DrainerConfig, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string, versionValue map[string]string) (*workingSet, error) {
	var err error

	var ws *workingSet

	instanceInProgress := firstInstanceInProgress(instances)
	if instanceInProgress != nil {
		return ws.WithInstanceAlreadyBeingUpdated(instanceInProgress), nil
	}

	instanceToUpdate := firstInstanceToUpdate(instances)
	if instanceToUpdate != nil {
		return ws.WithInstanceToUpdate(instanceToUpdate), nil
	}

	var instanceToReimage *compute.VirtualMachineScaleSetVM
	instanceToReimage, err = firstInstanceToReimage(customObject, instances, instanceNameFunc, versionValue)
	if err != nil {
		return ws, microerror.Mask(err)
	}
	if instanceToReimage != nil {
		instanceName := instanceNameFunc(customObject, *instanceToReimage.InstanceID)
		if isNodeDrained(drainerConfigs, instanceName) {
			return ws.WithInstanceToReimage(instanceToReimage), nil
		} else {
			return ws.WithInstanceToDrain(instanceToReimage), nil
		}
	}

	return ws, nil
}

// firstInstanceInProgress returns the first instance in the list not having a
// final state. In case all instances are in a final state
// firstInstanceInProgress returns nil.
func firstInstanceInProgress(list []compute.VirtualMachineScaleSetVM) *compute.VirtualMachineScaleSetVM {
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
func firstInstanceToReimage(customObject providerv1alpha1.AzureConfig, list []compute.VirtualMachineScaleSetVM, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string, versionValue map[string]string) (*compute.VirtualMachineScaleSetVM, error) {
	if versionValue == nil {
		return nil, microerror.Mask(versionBlobEmptyError)
	}

	for _, v := range list {
		desiredVersion := key.OperatorVersion(customObject)
		instanceName := instanceNameFunc(customObject, *v.InstanceID)
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
func firstInstanceToUpdate(list []compute.VirtualMachineScaleSetVM) *compute.VirtualMachineScaleSetVM {
	for _, v := range list {
		if *v.LatestModelApplied {
			continue
		}

		return &v
	}

	return nil
}

func isNodeDrained(drainerConfigs []corev1alpha1.DrainerConfig, instanceName string) bool {
	for _, n := range drainerConfigs {
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
