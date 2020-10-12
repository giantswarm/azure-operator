package nodepool

import (
	"context"

	apiextensionslabels "github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) waitForWorkersToBecomeReadyTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	cluster, err := util.GetClusterFromMetadata(ctx, r.CtrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if !cluster.GetDeletionTimestamp().IsZero() {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "Cluster is being deleted, skipping reconciling node pool")
		return currentState, nil
	}

	tenantClusterK8sClient, err := r.tenantClientFactory.GetClient(ctx, cluster)
	if err != nil {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant API not available yet", "stack", microerror.JSON(err))
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

		return currentState, nil
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "finding out if all tenant cluster worker nodes are Ready")

	readyForTransitioning, err := areNodesReadyForTransitioning(ctx, tenantClusterK8sClient, &azureMachinePool, isWorker)
	if IsClientNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, nil
	} else if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	if !readyForTransitioning {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster worker nodes are not Ready")
		return currentState, nil
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster worker nodes are Ready")

	return DrainOldWorkerNodes, nil
}

func countReadyNodes(ctx context.Context, tenantClusterK8sClient ctrlclient.Client, azureMachinePool *v1alpha3.AzureMachinePool, nodeRoleMatchFunc func(corev1.Node) bool) (int, error) {
	nodeList := &corev1.NodeList{}
	var labelSelector ctrlclient.MatchingLabels
	{
		labelSelector = make(map[string]string)
		labelSelector[apiextensionslabels.MachinePool] = azureMachinePool.Name
	}

	err := tenantClusterK8sClient.List(ctx, nodeList, labelSelector)
	if err != nil {
		return 0, microerror.Mask(err)
	}

	var numNodes int
	for _, n := range nodeList.Items {
		if nodeRoleMatchFunc(n) && isReady(n) {
			numNodes++
		}
	}

	return numNodes, nil
}

func areNodesReadyForTransitioning(ctx context.Context, tenantClusterK8sClient ctrlclient.Client, azureMachinePool *v1alpha3.AzureMachinePool, nodeRoleMatchFunc func(corev1.Node) bool) (bool, error) {
	numNodes, err := countReadyNodes(ctx, tenantClusterK8sClient, azureMachinePool, nodeRoleMatchFunc)
	if err != nil {
		return false, microerror.Mask(err)
	}

	// There must be at least one node registered for the cluster.
	if numNodes < 1 {
		return false, nil
	}

	return true, nil
}

func isWorker(n corev1.Node) bool {
	for k, v := range n.Labels {
		switch k {
		case "role":
			return v == "worker"
		case "kubernetes.io/role":
			return v == "worker"
		case "node-role.kubernetes.io/worker":
			return true
		case "node.kubernetes.io/worker":
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
