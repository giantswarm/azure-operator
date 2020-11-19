package nodepool

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) drainOldWorkerNodesTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "Cluster is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	virtualMachineScaleSetsClient, err := r.ClientFactory.GetVirtualMachineScaleSetsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, azureMachinePool.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	tenantClusterK8sClient, err := r.tenantClientFactory.GetClient(ctx, cluster)
	if tenantcluster.IsAPINotAvailableError(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available yet")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return currentState, nil
	} else if err != nil {
		return currentState, microerror.Mask(err)
	}

	vmss, err := virtualMachineScaleSetsClient.Get(ctx, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "finding all drainerconfigs")

	drainerConfigs := make(map[string]corev1alpha1.DrainerConfig)
	var labelSelector client.MatchingLabels
	{
		labelSelector = make(map[string]string)
		labelSelector[label.Cluster] = key.ClusterID(&azureMachinePool)
	}

	drainerConfigList := &corev1alpha1.DrainerConfigList{}
	err = r.CtrlClient.List(ctx, drainerConfigList, labelSelector, client.InNamespace(metav1.NamespaceAll))
	if err != nil {
		return "", microerror.Mask(err)
	}
	for _, dc := range drainerConfigList.Items {
		drainerConfigs[dc.Name] = dc
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d drainerconfigs", len(drainerConfigs)))
	r.Logger.LogCtx(ctx, "level", "debug", "message", "finding all worker VMSS instances")

	allWorkerInstances, err := r.GetVMSSInstances(ctx, virtualMachineScaleSetVMsClient, key.ClusterID(&azureMachinePool), key.NodePoolVMSSName(&azureMachinePool))
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d worker VMSS instances", len(allWorkerInstances)))
	r.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring that drainerconfig exists for all old worker nodes")

	var nodesPendingDraining int
	for _, i := range allWorkerInstances {
		old, err := r.isWorkerInstanceFromPreviousRelease(ctx, tenantClusterK8sClient, azureMachinePool.Name, i, vmss)
		if err != nil {
			return DeploymentUninitialized, nil
		}

		if old == nil || !*old {
			// Node is a new one or we weren't able to check it's status, don't drain it.
			continue
		}

		n := key.NodePoolInstanceName(azureMachinePool.Name, *i.InstanceID)

		dc, drainerConfigExists := drainerConfigs[n]
		if !drainerConfigExists {
			nodesPendingDraining++
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating drainerconfig for %s", n))
			err = r.CreateDrainerConfig(ctx, key.ClusterID(&azureMachinePool), cluster.Spec.ControlPlaneEndpoint.String(), key.NodePoolInstanceName(azureMachinePool.Name, *i.InstanceID))
			if err != nil {
				return DeploymentUninitialized, microerror.Mask(err)
			}
		}

		if drainerConfigExists && dc.Status.HasTimeoutCondition() {
			nodesPendingDraining++
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("drainerconfig for %s already exists but has timed out", n))
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting drainerconfig for %s", n))

			err = r.CtrlClient.Delete(ctx, &dc)
			if errors.IsNotFound(err) {
				r.Logger.LogCtx(ctx, "level", "debug", "message", "did not delete drainer config for tenant cluster node")
				r.Logger.LogCtx(ctx, "level", "debug", "message", "drainer config for tenant cluster node does not exist")
			} else if err != nil {
				return DeploymentUninitialized, microerror.Mask(err)
			}

			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating drainerconfig for %s", n))
			err = r.CreateDrainerConfig(ctx, key.ClusterID(&azureMachinePool), cluster.Spec.ControlPlaneEndpoint.String(), key.NodePoolInstanceName(azureMachinePool.Name, *i.InstanceID))
			if err != nil {
				return DeploymentUninitialized, microerror.Mask(err)
			}
		}

		if drainerConfigExists && !dc.Status.HasTimeoutCondition() && !dc.Status.HasDrainedCondition() {
			nodesPendingDraining++
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("drainerconfig for %s already exists", n))
		}
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "ensured that drainerconfig exists for all old worker nodes")
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%d nodes are pending draining", nodesPendingDraining))

	if nodesPendingDraining > 0 {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return currentState, nil
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "deleting all drainerconfigs")

	// Delete DrainerConfigs now that all nodes have been DRAINED.
	for _, dc := range drainerConfigs {
		err = r.CtrlClient.Delete(ctx, &dc)
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "deleted all drainerconfigs")

	return TerminateOldWorkerInstances, nil
}
