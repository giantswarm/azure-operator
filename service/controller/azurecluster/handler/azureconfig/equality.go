package azureconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"

	"github.com/giantswarm/azure-operator/v6/pkg/annotation"
)

func azureConfigsEqual(cr1, cr2 providerv1alpha1.AzureConfig) bool {
	// Common Cluster Checks.
	switch {
	case cr1.Spec.Cluster.Calico != cr2.Spec.Cluster.Calico:
		return false
	case cr1.Spec.Cluster.Customer != cr2.Spec.Cluster.Customer:
		return false
	case cr1.Spec.Cluster.Docker != cr2.Spec.Cluster.Docker:
		return false
	case cr1.Spec.Cluster.Etcd != cr2.Spec.Cluster.Etcd:
		return false
	case cr1.Spec.Cluster.ID != cr2.Spec.Cluster.ID:
		return false
	case !commonClusterKubernetesEqual(cr1.Spec.Cluster.Kubernetes, cr2.Spec.Cluster.Kubernetes):
		return false
	case !commonClusterNodesEqual(cr1.Spec.Cluster.Masters, cr2.Spec.Cluster.Masters):
		return false
	case cr1.Spec.Cluster.Scaling != cr2.Spec.Cluster.Scaling:
		return false
	case !commonClusterNodesEqual(cr1.Spec.Cluster.Workers, cr2.Spec.Cluster.Workers):
		return false
	}

	// Azure Provider Spec Checks.
	switch {
	case !intSliceEqual(cr1.Spec.Azure.AvailabilityZones, cr2.Spec.Azure.AvailabilityZones):
		return false
	case cr1.Spec.Azure.CredentialSecret != cr2.Spec.Azure.CredentialSecret:
		return false
	case cr1.Spec.Azure.DNSZones != cr2.Spec.Azure.DNSZones:
		return false
	case !azureClusterNodesEqual(cr1.Spec.Azure.Masters, cr2.Spec.Azure.Masters):
		return false
	//
	// cr.Spec.Azure.VirtualNetwork is omitted because it is managed by ipam resource.
	//
	case !azureClusterNodesEqual(cr1.Spec.Azure.Workers, cr2.Spec.Azure.Workers):
		return false
	}

	// Legacy version bundle version.
	if cr1.Spec.VersionBundle != cr2.Spec.VersionBundle { // nolint: gosimple
		return false
	}

	// External IP address for workers egress changed
	if cr1.Annotations[annotation.WorkersEgressExternalPublicIP] != cr2.Annotations[annotation.WorkersEgressExternalPublicIP] {
		return false
	}

	return true
}

func azureClusterNodesEqual(nodes1, nodes2 []providerv1alpha1.AzureConfigSpecAzureNode) bool {
	if len(nodes1) != len(nodes2) {
		return false
	}

	for i := 0; i < len(nodes1); i++ {
		if nodes1[i] != nodes2[i] {
			return false
		}
	}

	return true
}

func commonClusterKubernetesEqual(v1, v2 providerv1alpha1.ClusterKubernetes) bool {
	switch {
	case v1.API != v2.API:
		return false
	case v1.CloudProvider != v2.CloudProvider:
		return false
	case v1.DNS != v2.DNS:
		return false
	case v1.Domain != v2.Domain:
		return false
	case v1.IngressController != v2.IngressController:
		return false
	case v1.Kubelet != v2.Kubelet:
		return false
	case v1.NetworkSetup != v2.NetworkSetup:
		return false
	}

	return true
}

func commonClusterNodesEqual(nodes1, nodes2 []providerv1alpha1.ClusterNode) bool {
	if len(nodes1) != len(nodes2) {
		return false
	}

	for i := 0; i < len(nodes1); i++ {
		if nodes1[i].ID != nodes2[i].ID {
			return false
		}
	}

	return true
}

func intSliceEqual(xs1, xs2 []int) bool {
	if len(xs1) != len(xs2) {
		return false
	}

	for i := 0; i < len(xs1); i++ {
		if xs1[i] != xs2[i] {
			return false
		}
	}

	return true
}
