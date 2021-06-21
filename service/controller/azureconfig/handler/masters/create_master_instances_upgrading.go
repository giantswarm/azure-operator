package masters

import (
	"context"

	"github.com/giantswarm/errors/tenant"
	"github.com/giantswarm/tenantcluster/v3/pkg/tenantcluster"

	"github.com/giantswarm/azure-operator/v5/pkg/label"

	"github.com/giantswarm/azure-operator/v5/pkg/project"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes"
	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) masterInstancesUpgradingTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// We have a weird race condition somewhere that makes this state to be applied when it still contains old
	// configuration values like the cloudconfig blob. To work around it we check if the masters VMSS is
	// up to date.
	isMastersVmssUpToDate, err := r.isMastersVmssUpToDate(ctx, &cr)
	if err != nil || !isMastersVmssUpToDate {
		return "", nil
	}

	var tenantClusterK8sClient client.Client
	{
		tenantClusterK8sClient, err = r.getTenantClusterClient(ctx, &cr)
		if tenant.IsAPINotAvailable(err) || tenantcluster.IsTimeout(err) {
			// The kubernetes API is not reachable. This usually happens when a new cluster is being created.
			// This makes the whole controller to fail and stops next handlers from being executed even if they are
			// safe to run. We don't want that to happen so we just return and we'll try again during next loop.
			r.Logger.Debugf(ctx, "tenant API not available yet")
			r.Logger.Debugf(ctx, "canceling resource")

			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		}
	}

	r.Logger.Debugf(ctx, "finding out if all tenant cluster master nodes are Ready")

	tenantNodes, err := r.getTenantClusterNodes(ctx, tenantClusterK8sClient)
	if IsClientNotFound(err) {
		r.Logger.Debugf(ctx, "tenant cluster client not available yet")
		return currentState, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if !areNodesReadyForUpgrading(tenantNodes) {
		r.Logger.Debugf(ctx, "found out that at least one master node is not ready")
		return currentState, nil
	}

	r.Logger.Debugf(ctx, "found out that all tenant cluster master nodes are Ready")

	versionValue := map[string]string{}
	for i, node := range tenantNodes {
		versionValue[node.Name] = key.ReleaseVersion(&tenantNodes[i])
	}

	var masterUpgradeInProgress bool
	{
		allMasterInstances, err := r.allInstances(ctx, cr, key.MasterVMSSName)
		if nodes.IsScaleSetNotFound(err) {
			r.Logger.Debugf(ctx, "did not find the scale set '%s'", key.MasterVMSSName(cr))
		} else if err != nil {
			return "", microerror.Mask(err)
		} else {
			r.Logger.Debugf(ctx, "processing master VMSSs")

			// Ensure that all VM instances are in Successful state before proceeding with reimaging.
			for _, vm := range allMasterInstances {
				if vm.ProvisioningState != nil && !key.IsSucceededProvisioningState(*vm.ProvisioningState) {
					r.Logger.Debugf(ctx, "master instance %#q is not in successful provisioning state: %#q", key.MasterInstanceName(cr, *vm.InstanceID), *vm.ProvisioningState)
					r.Logger.Debugf(ctx, "cancelling resource")
					return currentState, nil
				}
			}

			desiredVersion := key.ReleaseVersion(&cr)
			for i, vm := range allMasterInstances {
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
					err = r.updateInstance(ctx, cr, &allMasterInstances[i], key.MasterVMSSName, key.MasterInstanceName)
					if err != nil {
						return "", microerror.Mask(err)
					}

					// Update only one instance at at time.
					break
				}

				// Once the VM instance configuration has been updated, it can be reimaged.
				err = r.reimageInstance(ctx, cr, &allMasterInstances[i], key.MasterVMSSName, key.MasterInstanceName)
				if err != nil {
					return "", microerror.Mask(err)
				}

				// Reimage only one instance at a time.
				break
			}

			r.Logger.Debugf(ctx, "processed master VMSSs")
		}
	}

	if !masterUpgradeInProgress {
		// When masters are upgraded, consider the process to be completed.
		return DeploymentCompleted, nil
	}

	// Upgrade still in progress. Keep current state.
	return currentState, nil
}

// isMastersVmssUpToDate checks whether or not the masters VMSS has been updated. We rely on the tag containing the az-op
// version, as we are mostly interested on the cloudconfig blob, which depends on the az-op version.
func (r *Resource) isMastersVmssUpToDate(ctx context.Context, azureConfig *providerv1alpha1.AzureConfig) (bool, error) {
	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureConfig.ObjectMeta)
	if err != nil {
		return false, microerror.Mask(err)
	}

	mastersVMSS, err := virtualMachineScaleSetsClient.Get(ctx, key.ClusterID(azureConfig), key.MasterVMSSName(*azureConfig))
	if err != nil {
		return false, microerror.Mask(err)
	}

	azureOperatorVersionTag, ok := mastersVMSS.Tags[label.AzureOperatorVersionTag]
	if !ok || *azureOperatorVersionTag != project.Version() {
		return false, nil
	}

	return true, nil
}

func (r *Resource) getTenantClusterNodes(ctx context.Context, tenantClusterK8sClient client.Client) ([]corev1.Node, error) {
	nodeList := &corev1.NodeList{}
	err := tenantClusterK8sClient.List(ctx, nodeList)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return nodeList.Items, nil
}

func areNodesReadyForUpgrading(nodes []corev1.Node) bool {
	var numNodes int
	for _, n := range nodes {
		if isMaster(n) {
			numNodes++

			if !isReady(n) {
				// If there's even one node that is not ready, then wait.
				return false
			}
		}
	}

	// There must be at least one node registered for the cluster.
	return numNodes > 0
}

func (r *Resource) reimageInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	instanceName := instanceNameFunc(customObject, *instance.InstanceID)

	r.Logger.Debugf(ctx, "ensuring instance '%s' to be reimaged", instanceName)

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

	r.Logger.Debugf(ctx, "ensured instance '%s' to be reimaged", instanceName)

	return nil
}

func (r *Resource) updateInstance(ctx context.Context, customObject providerv1alpha1.AzureConfig, instance *compute.VirtualMachineScaleSetVM, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, instanceNameFunc func(customObject providerv1alpha1.AzureConfig, instanceID string) string) error {
	if instance == nil {
		return nil
	}

	instanceName := instanceNameFunc(customObject, *instance.InstanceID)

	r.Logger.Debugf(ctx, "ensuring instance '%s' to be updated", instanceName)

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

	r.Logger.Debugf(ctx, "ensured instance '%s' to be updated", instanceName)

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
