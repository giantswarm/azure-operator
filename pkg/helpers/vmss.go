package helpers

import (
	"context"

	"github.com/coreos/go-semver/semver"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/pkg/label"
	"github.com/giantswarm/azure-operator/pkg/project"
	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
)

func IsClusterCreating(cr providerv1alpha1.AzureConfig) bool {
	// When cluster creation is in the beginning, it doesn't necessarily have
	// any status conditions yet.
	if len(cr.Status.Cluster.Conditions) == 0 {
		return true
	}
	if cr.Status.Cluster.HasCreatingCondition() {
		return true
	}

	return false
}

// AnyNodesOutOfDate iterates over all nodes in tenant cluster and finds
// corresponding azure-operator version from node labels. If node doesn't have
// this label or was created with older version than currently reconciling one,
// then this function returns true. Otherwise (including on error) false.
func AnyNodesOutOfDate(ctx context.Context) (bool, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		return false, clientNotFoundError
	}

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return false, microerror.Mask(err)
	}

	myVersion := semver.New(project.Version())
	for _, n := range nodeList.Items {
		v, exists := n.GetLabels()[label.OperatorVersion]
		if !exists {
			return true, nil
		}

		nodeVersion := semver.New(v)

		if nodeVersion.LessThan(*myVersion) {
			return true, nil
		}
	}

	return false, nil
}

func AreMasterNodesReadyForTransitioning(ctx context.Context) (bool, error) {
	return areNodesReadyForTransitioning(ctx, isMaster)
}

func AreWorkerNodesReadyForTransitioning(ctx context.Context) (bool, error) {
	return areNodesReadyForTransitioning(ctx, isWorker)
}

func areNodesReadyForTransitioning(ctx context.Context, nodeRoleMatchFunc func(corev1.Node) bool) (bool, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		return false, clientNotFoundError
	}

	nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
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
		if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue && c.Reason == "KubeletReady" {
			return true
		}
	}

	return false
}
