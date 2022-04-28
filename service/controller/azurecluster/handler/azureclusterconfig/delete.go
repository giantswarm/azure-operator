package azureclusterconfig

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/v6/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/finalizerskeptcontext"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	azureCluster, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensuring AzureClusterConfig deletion")

	var azureClusterConfig corev1alpha1.AzureClusterConfig
	{
		nsName := types.NamespacedName{
			Name:      clusterConfigName(key.ClusterID(&azureCluster)),
			Namespace: metav1.NamespaceDefault,
		}
		err = r.ctrlClient.Get(ctx, nsName, &azureClusterConfig)
		if errors.IsNotFound(err) {
			// Done. AzureClusterConfig is gone and finalizer can be released.
			r.logger.Debugf(ctx, "AzureClusterConfig deleted")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	// Wait until AzureClusterConfig is gone.
	finalizerskeptcontext.SetKept(ctx)

	if key.IsDeleted(&azureClusterConfig) {
		r.logger.Debugf(ctx, "AzureClusterConfig deletion in progress")

		err = r.ensureFinalizersRemoved(ctx, azureClusterConfig)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "canceling resource")
		return nil
	}

	err = r.ctrlClient.Delete(ctx, &azureClusterConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured AzureClusterConfig deletion")

	err = r.ensureFinalizersRemoved(ctx, azureClusterConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureFinalizersRemoved(ctx context.Context, azureClusterConfig corev1alpha1.AzureClusterConfig) error {
	// There is no operator reconciling AzureClusterConfig CR any more.
	// I remove all finalizers to avoid deletion being stuck.
	if len(azureClusterConfig.Finalizers) > 0 {
		r.logger.Debugf(ctx, "clearing finalizers from AzureClusterConfig")
		patch := client.MergeFrom(azureClusterConfig.DeepCopy())
		azureClusterConfig.Finalizers = make([]string, 0)
		err := r.ctrlClient.Patch(ctx, &azureClusterConfig, patch)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "cleared finalizers from AzureClusterConfig")
	}

	return nil
}
