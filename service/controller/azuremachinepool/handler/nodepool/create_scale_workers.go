package nodepool

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/coreos/go-semver/semver"
	apiextensionslabels "github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/scalestrategy"
	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// The goal of scaleUpWorkerVMSSTransition is to double the desired number
// of nodes in worker VMSS in order to provide 1:1 mapping between new
// up-to-date nodes when draining and terminating old nodes.
// This will be done in subsequent reconciliation loops to avoid hitting the
// VMSS api too hard.
func (r *Resource) scaleUpWorkerVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	machinePool, err := r.getOwnerMachinePool(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if machinePool == nil {
		return currentState, microerror.Mask(ownerReferenceNotSet)
	}

	if !machinePool.GetDeletionTimestamp().IsZero() {
		r.Logger.Debugf(ctx, "MachinePool is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	allReady, err := vmsscheck.InstancesAreRunning(ctx, r.Logger, virtualMachineScaleSetVMsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	// Not all workers are Running in Azure, wait for next reconciliation loop.
	if !allReady {
		return currentState, nil
	}

	strategy := scalestrategy.Quick{}

	// Ensure the deployment is successful before we move on with scaling.
	currentDeployment, err := deploymentsClient.Get(ctx, key.ClusterID(&azureMachinePool), key.NodePoolDeploymentName(&azureMachinePool))
	if IsDeploymentNotFound(err) {
		// Deployment not found, we need to apply it again.
		return DeploymentUninitialized, microerror.Mask(err)
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	switch *currentDeployment.Properties.ProvisioningState {
	case "Failed", "Canceled":
		// Deployment is failed or canceled, I need to go back and re-apply it.
		r.Logger.Debugf(ctx, "Node Pool deployment is in state %s, we need to reapply it.", *currentDeployment.Properties.ProvisioningState)
		return DeploymentUninitialized, nil
	case "Succeeded":
		// Deployment is succeeded, safe to go on.
	default:
		// Deployment is still running, we need to wait for another reconciliation loop.
		r.Logger.Debugf(ctx, "Node Pool deployment is in state %s, waiting for it to be succeeded.", *currentDeployment.Properties.ProvisioningState)
		return currentState, nil
	}

	oldInstances, newInstances, err := r.splitInstancesByUpdatedStatus(ctx, azureMachinePool)
	if tenantcluster.IsAPINotAvailableError(err) {
		r.Logger.Debugf(ctx, "tenant API not available yet")
		r.Logger.Debugf(ctx, "canceling resource")

		return currentState, nil
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	desiredWorkerCount := int64(len(oldInstances) * 2)
	r.Logger.Debugf(ctx, "The desired number of workers is: %d", desiredWorkerCount)

	if desiredWorkerCount > int64(len(oldInstances)+len(newInstances)) {
		// Disable cluster autoscaler for this nodepool.
		err = r.disableClusterAutoscaler(ctx, azureMachinePool)
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}

		err = r.ScaleVMSS(ctx, azureMachinePool, desiredWorkerCount, strategy)
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}

		r.Logger.Debugf(ctx, "scaled worker VMSS to %d nodes", desiredWorkerCount)

		// Let's stay in the current state.
		return currentState, nil
	}

	// We didn't scale up the VMSS, ready to move to next step.
	return WaitForWorkersToBecomeReady, nil
}

func (r *Resource) splitInstancesByUpdatedStatus(ctx context.Context, azureMachinePool capzv1alpha3.AzureMachinePool) ([]compute.VirtualMachineScaleSetVM, []compute.VirtualMachineScaleSetVM, error) {
	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.Logger.Debugf(ctx, "Cluster is being deleted, skipping reconciling node pool")
		return nil, nil, nil
	}

	// All workers ready, we can scale up if needed.
	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		r.Logger.Debugf(ctx, "finding all worker VMSS instances")

		allWorkerInstances, err = r.GetVMSSInstances(ctx, azureMachinePool)
		if err != nil {
			return nil, nil, microerror.Mask(err)
		}

		r.Logger.Debugf(ctx, "found %d worker VMSS instances", len(allWorkerInstances))
	}

	resourceGroup := key.ClusterID(&azureMachinePool)
	vmssName := key.NodePoolVMSSName(&azureMachinePool)

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	vmss, err := virtualMachineScaleSetsClient.Get(ctx, resourceGroup, vmssName)
	if err != nil {
		return nil, nil, microerror.Mask(err)
	}

	var oldInstances []compute.VirtualMachineScaleSetVM
	var newInstances []compute.VirtualMachineScaleSetVM
	{
		for _, i := range allWorkerInstances {
			old, err := r.isWorkerInstanceFromPreviousRelease(ctx, cluster, azureMachinePool.Name, i)
			if err != nil {
				return nil, nil, microerror.Mask(err)
			}

			skuChanged := *i.Sku.Name != *vmss.Sku.Name

			if old || skuChanged {
				oldInstances = append(oldInstances, i)
			} else {
				newInstances = append(newInstances, i)
			}
		}
	}

	return oldInstances, newInstances, nil
}

func (r *Resource) getK8sWorkerNodeForInstance(ctx context.Context, tenantClusterK8sClient ctrlclient.Client, nodePoolId string, instance compute.VirtualMachineScaleSetVM) (*corev1.Node, error) {
	name := key.NodePoolInstanceName(nodePoolId, *instance.InstanceID)

	nodeList := &corev1.NodeList{}
	labelSelector := ctrlclient.MatchingLabels{apiextensionslabels.MachinePool: nodePoolId}
	err := tenantClusterK8sClient.List(ctx, nodeList, labelSelector)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	nodes := nodeList.Items
	for _, n := range nodes {
		if n.GetName() == name {
			return &n, nil
		}
	}

	// Node related to this instance was not found.
	return nil, nil
}

func (r *Resource) isWorkerInstanceFromPreviousRelease(ctx context.Context, cluster *capiv1alpha3.Cluster, nodePoolId string, instance compute.VirtualMachineScaleSetVM) (bool, error) {
	tenantClusterK8sClient, err := r.tenantClientFactory.GetClient(ctx, cluster)
	if err != nil {
		return false, microerror.Mask(err)
	}

	n, err := r.getK8sWorkerNodeForInstance(ctx, tenantClusterK8sClient, nodePoolId, instance)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if n == nil {
		// Kubernetes node related to this instance not found, we consider the node old.
		return true, nil
	}

	myVersion := semver.New(project.Version())

	v, exists := n.GetLabels()[label.OperatorVersion]
	if !exists {
		// Label does not exist, this normally happens when a new node is coming up but did not finish
		// its kubernetes bootstrap yet and thus doesn't have all the needed labels.
		// We'll ignore this node for now and wait for it to bootstrap correctly.
		return false, nil
	}

	nodeVersion := semver.New(v)
	if nodeVersion.LessThan(*myVersion) {
		return true, nil
	}

	return false, nil
}
