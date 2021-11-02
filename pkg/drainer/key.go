package drainer

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func isCriticalPod(podName string) bool {
	r := false
	// k8s-api-healthz is a service on master nodes that exposes
	// unauthenticated apiserver /healthz for load balancers. It is deployed as
	// manifest similar to api-server, controller-manager and scheduler and
	// therefore it always restarts after termination.
	r = r || strings.HasPrefix(podName, "k8s-api-healthz")
	r = r || strings.HasPrefix(podName, "k8s-api-server")
	r = r || strings.HasPrefix(podName, "k8s-controller-manager")
	r = r || strings.HasPrefix(podName, "k8s-scheduler")

	return r
}

func isDaemonSetPod(pod v1.Pod) bool {
	r := false
	ownerRefrence := metav1.GetControllerOf(&pod)

	if ownerRefrence != nil && ownerRefrence.Kind == "DaemonSet" {
		r = true
	}

	return r
}

func isEvictedPod(pod v1.Pod) bool {
	return pod.Status.Reason == "Evicted"
}
