package azureclusterconfig

import corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"

func azureClusterConfigsEqual(cr1, cr2 corev1alpha1.AzureClusterConfig) bool {
	switch {
	case !clusterGuestConfigEqual(cr1.Spec.Guest.ClusterGuestConfig, cr2.Spec.Guest.ClusterGuestConfig):
		return false
	case cr1.Spec.Guest.CredentialSecret != cr2.Spec.Guest.CredentialSecret:
		return false
	case !masterNodesEqual(cr1.Spec.Guest.Masters, cr2.Spec.Guest.Masters):
		return false
	case !workerNodesEqual(cr1.Spec.Guest.Workers, cr2.Spec.Guest.Workers):
		return false
	case cr1.Spec.VersionBundle != cr2.Spec.VersionBundle:
		return false
	}

	return true
}

func clusterGuestConfigEqual(cfg1, cfg2 corev1alpha1.ClusterGuestConfig) bool {
	switch {
	case cfg1.AvailabilityZones != cfg2.AvailabilityZones:
		return false
	case cfg1.DNSZone != cfg2.DNSZone:
		return false
	case cfg1.ID != cfg2.ID:
		return false
	case cfg1.Name != cfg2.Name:
		return false
	case cfg1.Owner != cfg2.Owner:
		return false
	case cfg1.ReleaseVersion != cfg2.ReleaseVersion:
		return false
	case !versionBundlesEqual(cfg1.VersionBundles, cfg2.VersionBundles):
		return false

	}

	return true
}

func masterNodesEqual(nodes1, nodes2 []corev1alpha1.AzureClusterConfigSpecGuestMaster) bool {
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

func workerNodesEqual(nodes1, nodes2 []corev1alpha1.AzureClusterConfigSpecGuestWorker) bool {
	if len(nodes1) != len(nodes2) {
		return false
	}

	for i := 0; i < len(nodes1); i++ {
		if nodes1[i].ID != nodes2[i].ID || nodes1[i].VMSize != nodes2[i].VMSize {
			return false
		}

		if len(nodes1[i].Labels) != len(nodes2[i].Labels) {
			return false
		}

		for k, v := range nodes1[i].Labels {
			v2, exists := nodes2[i].Labels[k]
			if !exists || v != v2 {
				return false
			}
		}
	}

	return true
}

func versionBundlesEqual(vbs1, vbs2 []corev1alpha1.ClusterGuestConfigVersionBundle) bool {
	if len(vbs1) != len(vbs2) {
		return false
	}

	for i := 0; i < len(vbs1); i++ {
		if vbs1[i] != vbs2[i] {
			return false
		}
	}

	return true
}
