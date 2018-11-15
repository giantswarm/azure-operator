package instance

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Stage = "Stage"
)

const (
	DeploymentInitialized  = "DeploymentInitialized"
	InstancesUpgrading     = "InstancesUpgrading"
	ProvisioningSuccessful = "ProvisioningSuccessful"
)

func (r *Resource) deleteResourceStatus(customObject providerv1alpha1.AzureConfig, t string, s string) error {
	customObject = computeForDeleteResourceStatus(customObject, t, s)

	n := customObject.GetNamespace()
	_, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(n).UpdateStatus(&customObject)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) setResourceStatus(customObject providerv1alpha1.AzureConfig, t string, s string) error {
	// Get the newest CR version. Otherwise status update may fail because of:
	//
	//	 the object has been modified; please apply your changes to the
	//	 latest version and try again
	//
	{
		c, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(customObject.Namespace).Get(customObject.Name, metav1.GetOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		customObject = *c
	}

	resourceStatus := providerv1alpha1.StatusClusterResource{
		Conditions: []providerv1alpha1.StatusClusterResourceCondition{
			{
				Status: s,
				Type:   t,
			},
		},
		Name: Name,
	}

	var set bool
	for i, r := range customObject.Status.Cluster.Resources {
		if r.Name != Name {
			continue
		}

		for _, c := range r.Conditions {
			if c.Type == t {
				continue
			}
			resourceStatus.Conditions = append(resourceStatus.Conditions, c)
		}

		customObject.Status.Cluster.Resources[i] = resourceStatus
		set = true
	}

	if !set {
		customObject.Status.Cluster.Resources = append(customObject.Status.Cluster.Resources, resourceStatus)
	}

	{
		n := customObject.GetNamespace()
		_, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(n).UpdateStatus(&customObject)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func computeForDeleteResourceStatus(customObject providerv1alpha1.AzureConfig, t string, s string) providerv1alpha1.AzureConfig {
	var cleanup bool
	{
		resourceStatus := providerv1alpha1.StatusClusterResource{
			Conditions: nil,
			Name:       Name,
		}

		for i, r := range customObject.Status.Cluster.Resources {
			if r.Name != Name {
				continue
			}

			for _, c := range r.Conditions {
				if c.Type == t && c.Status == s {
					continue
				}
				resourceStatus.Conditions = append(resourceStatus.Conditions, c)
			}

			if len(resourceStatus.Conditions) > 0 {
				customObject.Status.Cluster.Resources[i] = resourceStatus
			} else {
				cleanup = true
			}

			break
		}
	}

	if cleanup {
		var list []providerv1alpha1.StatusClusterResource
		for _, r := range customObject.Status.Cluster.Resources {
			if r.Name == Name {
				continue
			}
			list = append(list, r)
		}
		customObject.Status.Cluster.Resources = list
	}

	return customObject
}

func hasResourceStatus(customObject providerv1alpha1.AzureConfig, t string, s string) bool {
	for _, r := range customObject.Status.Cluster.Resources {
		if r.Name != Name {
			continue
		}

		for _, c := range r.Conditions {
			if c.Type == t && c.Status == s {
				return true
			}
		}
	}

	return false
}

func resourceStatusExists(customObject providerv1alpha1.AzureConfig, t string) bool {
	for _, r := range customObject.Status.Cluster.Resources {
		if r.Name != Name {
			continue
		}

		for _, c := range r.Conditions {
			if c.Type == t {
				return true
			}
		}
	}

	return false
}
