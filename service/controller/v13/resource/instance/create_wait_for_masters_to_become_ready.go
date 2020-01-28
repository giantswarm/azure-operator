package instance

import (
	"context"

	"github.com/giantswarm/azure-operator/service/controller/v13/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) waitForMastersToBecomeReadyTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return "", microerror.Mask(err)
	}

	var numMasters int
	for _, n := range nodeList.Items {
		if isMaster(n) {
			numMasters++

			if !isReady(n) {
				// If there's even one master that is not ready, then wait.
				return currentState, nil
			}
		}
	}

	// There must be at least one master registered for the cluster.
	if numMasters < 1 {
		return currentState, nil
	}

	return ScaleUpWorkerVMSS, nil
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
		if c.Type == corev1.NodeReady {
			return true
		}
	}

	return false
}
