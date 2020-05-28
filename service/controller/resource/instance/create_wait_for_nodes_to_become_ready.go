package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
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

	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// If the old VMSS still exists, we want to go to a different state.
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Checking if the legacy VMSS %s is still present", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck
	vmss, err := r.getScaleSet(ctx, cr, key.ResourceGroupName(cr), key.LegacyWorkerVMSSName(cr))
	if IsScaleSetNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s was not found", key.LegacyWorkerVMSSName(cr)))
		return DrainOldWorkerNodes, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s is still present", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck

	// The legacy VMSS was found, check the scaling.
	if *vmss.Sku.Capacity > 0 {
		// The legacy VMSS has still instances running, cordon all of them.
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s has %d instances: draining those", key.LegacyWorkerVMSSName(cr), *vmss.Sku.Capacity)) // nolint: errcheck
		return DrainOldVMSS, nil
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s has 0 instances", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck

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

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
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
