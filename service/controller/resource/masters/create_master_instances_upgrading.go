package masters

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/nodes"
)

func (r *Resource) masterInstancesUpgradingTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "finding out if all tenant cluster master nodes are Ready")
	{
		allMasterNodesAreReady, err := r.areNodesReadyForUpgrading(ctx)
		if IsClientNotFound(err) {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		}

		if !allMasterNodesAreReady {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "found out that at least one master node is not ready")
			return currentState, nil
		}
	}
	r.Logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster master nodes are Ready")

	versionValue := map[string]string{}
	{
		for _, node := range cr.Status.Cluster.Nodes {
			versionValue[node.Name] = node.Version
		}
	}

	var masterUpgradeInProgress bool
	{
		allMasterInstances, err := r.AllInstances(ctx, cr, key.MasterVMSSName)
		if nodes.IsScaleSetNotFound(err) {
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.MasterVMSSName(cr)))
		} else if err != nil {
			return "", microerror.Mask(err)
		} else {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "processing master VMSSs")

			// Ensure that all VM instances are in Successful state before proceeding with reimaging.
			for _, vm := range allMasterInstances {
				if vm.ProvisioningState != nil && !key.IsSucceededProvisioningState(*vm.ProvisioningState) {
					r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("master instance %#q is not in successful provisioning state: %#q", key.MasterInstanceName(cr, *vm.InstanceID), *vm.ProvisioningState))
					r.Logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
					return currentState, nil
				}
			}

			desiredVersion := project.Version()
			for _, vm := range allMasterInstances {
				instanceName := key.MasterInstanceName(cr, *vm.InstanceID)
				instanceVersion, ok := versionValue[instanceName]
				if !ok {
					continue
				}
				if desiredVersion == instanceVersion {
					continue
				}

				masterUpgradeInProgress = true

				// Ensure that VM has latest VMSS configuration (includes ignition template etc.).
				if !*vm.VirtualMachineScaleSetVMProperties.LatestModelApplied {
					err = r.updateInstance(ctx, cr, &vm, key.MasterVMSSName, key.MasterInstanceName)
					if err != nil {
						return "", microerror.Mask(err)
					}

					// Update only one instance at at time.
					break
				}

				// Once the VM instance configuration has been updated, it can be reimaged.
				err = r.reimageInstance(ctx, cr, &vm, key.MasterVMSSName, key.MasterInstanceName)
				if err != nil {
					return "", microerror.Mask(err)
				}

				// Reimage only one instance at a time.
				break
			}

			r.Logger.LogCtx(ctx, "level", "debug", "message", "processed master VMSSs")
		}
	}

	if !masterUpgradeInProgress {
		// When masters are upgraded, consider the process to be completed.
		return DeploymentCompleted, nil
	}

	// Upgrade still in progress. Keep current state.
	return currentState, nil
}

func (r *Resource) areNodesReadyForUpgrading(ctx context.Context) (bool, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		return false, clientNotFoundError
	}

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	var numNodes int
	for _, n := range nodeList.Items {
		if isMaster(n) {
			numNodes++

			if !isReady(n) {
				// If there's even one node that is not ready, then wait.
				return false, nil
			}
		}
	}

	// There must be at least one node registered for the cluster.
	if numNodes < 1 {
		return false, nil
	}

	return true, nil
}

func (r *Resource) reimageInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	instanceName := instanceNameFunc(customObject, *instance.InstanceID)

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be reimaged", instanceName))

	c, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, customObject.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroupName := key.ResourceGroupName(customObject)
	vmssName := deploymentNameFunc(customObject)
	ids := &compute.VirtualMachineScaleSetReimageParameters{
		InstanceIds: to.StringSlicePtr([]string{
			*instance.InstanceID,
		}),
	}
	res, err := c.Reimage(ctx, resourceGroupName, vmssName, ids)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = c.ReimageResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be reimaged", instanceName))

	return nil
}

func (r *Resource) updateInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	instanceName := instanceNameFunc(customObject, *instance.InstanceID)

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring instance '%s' to be updated", instanceName))

	c, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, customObject.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroupName := key.ResourceGroupName(customObject)
	vmssName := deploymentNameFunc(customObject)
	ids := compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
		InstanceIds: to.StringSlicePtr([]string{
			*instance.InstanceID,
		}),
	}
	res, err := c.UpdateInstances(ctx, resourceGroupName, vmssName, ids)
	if err != nil {
		return microerror.Mask(err)
	}
	_, err = c.UpdateInstancesResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured instance '%s' to be updated", instanceName))

	return nil
}

func isMaster(n corev1.Node) bool {
	for k, v := range n.Labels {
		switch k {
		case "role":
			return v == "master"
		case "kubernetes.io/role":
			return v == "master"
		case "node-role.kubernetes.io/master":
			return true
		case "node.kubernetes.io/master":
			return true
		}
	}

	return false
}

func isReady(n corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue && c.Reason == "KubeletReady" {
			return true
		}
	}

	return false
}
