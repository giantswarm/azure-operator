package instance

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
)

func (r *Resource) waitForWorkersToBecomeReadyTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.Logger.LogCtx(ctx, "level", "debug", "message", "finding out if all tenant cluster worker nodes are Ready")

	readyForTransitioning, err := areNodesReadyForTransitioning(ctx, isWorker)
	if IsClientNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if !readyForTransitioning {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster worker nodes are not Ready")
		return currentState, nil
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster worker nodes are Ready")

	return DrainOldWorkerNodes, nil
}

func countReadyNodes(ctx context.Context, nodeRoleMatchFunc func(corev1.Node) bool) (int, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return 0, microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		return 0, clientNotFoundError
	}

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
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

func areNodesReadyForTransitioning(ctx context.Context, nodeRoleMatchFunc func(corev1.Node) bool) (bool, error) {
	numNodes, err := countReadyNodes(ctx, nodeRoleMatchFunc)
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
