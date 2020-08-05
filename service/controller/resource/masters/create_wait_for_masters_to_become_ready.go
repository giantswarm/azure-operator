package masters

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
)

func (r *Resource) waitForMastersToBecomeReadyTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.Logger.LogCtx(ctx, "level", "debug", "message", "finding out if all tenant cluster master nodes are Ready")

	readyForTransitioning, err := areNodesReadyForTransitioning(ctx, isMaster)
	if IsClientNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if !readyForTransitioning {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster master nodes are not Ready")
		return currentState, nil
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "found out that all tenant cluster master nodes are Ready")

	return DeploymentCompleted, nil
}

func areNodesReadyForTransitioning(ctx context.Context, nodeRoleMatchFunc func(corev1.Node) bool) (bool, error) {
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

func isReady(n corev1.Node) bool {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue && c.Reason == "KubeletReady" {
			return true
		}
	}

	return false
}
