package nodes

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) GetResourceStatus(ctx context.Context, customObject providerv1alpha1.AzureConfig, t string) (string, error) {
	{
		c, err := r.G8sClient.ProviderV1alpha1().AzureConfigs(customObject.Namespace).Get(ctx, customObject.Name, metav1.GetOptions{})
		if err != nil {
			return "", microerror.Mask(err)
		}

		customObject = *c
	}

	for _, resource := range customObject.Status.Cluster.Resources {
		if resource.Name != r.Name() {
			continue
		}

		for _, c := range resource.Conditions {
			if c.Type == t {
				return c.Status, nil
			}
		}
	}

	return "", nil
}

func (r *Resource) SetResourceStatus(ctx context.Context, customObject providerv1alpha1.AzureConfig, t string, s string) error {
	// Get the newest CR version. Otherwise status update may fail because of:
	//
	//	 the object has been modified; please apply your changes to the
	//	 latest version and try again
	//
	{
		c, err := r.G8sClient.ProviderV1alpha1().AzureConfigs(customObject.Namespace).Get(ctx, customObject.Name, metav1.GetOptions{})
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
		Name: r.Name(),
	}

	var set bool
	for i, resource := range customObject.Status.Cluster.Resources {
		if resource.Name != r.Name() {
			continue
		}

		for _, c := range resource.Conditions {
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
		_, err := r.G8sClient.ProviderV1alpha1().AzureConfigs(n).UpdateStatus(ctx, &customObject, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
