package instance

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
)

func (r *Resource) waitForMastersToBecomeReadyTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if all tenant cluster master nodes are Ready")

	tenantK8sClient, err := r.getK8sClient(ctx, obj)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, nil
	}

	readyForTransitioning, err := areNodesReadyForTransitioning(tenantK8sClient, isMaster)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if !readyForTransitioning {
		r.logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster master nodes are not Ready")
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster master nodes are Ready")

	return ScaleUpWorkerVMSS, nil
}

func (r *Resource) waitForWorkersToBecomeReadyTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if all tenant cluster worker nodes are Ready")

	tenantK8sClient, err := r.getK8sClient(ctx, obj)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, microerror.Mask(err)
	}

	readyForTransitioning, err := areNodesReadyForTransitioning(tenantK8sClient, isWorker)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if !readyForTransitioning {
		r.logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster worker nodes are not Ready")
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster worker nodes are Ready")

	return DrainOldWorkerNodes, nil
}

func areNodesReadyForTransitioning(tenantK8sClient kubernetes.Interface, nodeRoleMatchFunc func(corev1.Node) bool) (bool, error) {
	nodeList, err := tenantK8sClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	var numNodes int
	for _, n := range nodeList.Items {
		if nodeRoleMatchFunc(n) {
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
		if c.Type == corev1.NodeReady {
			return true
		}
	}

	return false
}
