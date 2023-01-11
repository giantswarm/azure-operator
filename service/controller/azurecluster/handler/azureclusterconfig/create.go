package azureclusterconfig

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/azure-operator/v7/pkg/label"
	"github.com/giantswarm/azure-operator/v7/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	azureCluster, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var azureClusterConfig corev1alpha1.AzureClusterConfig
	{
		nsName := types.NamespacedName{
			Name:      clusterConfigName(key.ClusterName(&azureCluster)),
			Namespace: metav1.NamespaceDefault,
		}
		err = r.ctrlClient.Get(ctx, nsName, &azureClusterConfig)
		if errors.IsNotFound(err) {
			r.logger.Debugf(ctx, "azureclusterconfig did not exist, nothing to do")
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.Debugf(ctx, "ensuring azureclusterconfig's %q label is up to date", label.ClusterOperatorVersion)
	{
		currentClusterOperatorVersion := azureClusterConfig.GetLabels()[label.ClusterOperatorVersion]
		desiredClusterOperatorVersion := azureCluster.GetLabels()[label.ClusterOperatorVersion]

		if currentClusterOperatorVersion != desiredClusterOperatorVersion {
			r.logger.Debugf(ctx, "updating %q label on azureclusterconfig", label.ClusterOperatorVersion)
			azureClusterConfig.Labels[label.ClusterOperatorVersion] = desiredClusterOperatorVersion

			err = r.ctrlClient.Update(ctx, &azureClusterConfig)
			if err != nil {
				return microerror.Mask(err)
			}
			r.logger.Debugf(ctx, "updated %q label on azureclusterconfig", label.ClusterOperatorVersion)
		}
	}

	return nil
}

func clusterConfigName(clusterID string) string {
	return fmt.Sprintf("%s-%s", clusterID, "azure-cluster-config")
}
