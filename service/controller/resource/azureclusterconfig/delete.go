package azureclusterconfig

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/finalizerskeptcontext"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	azureCluster, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureClusterConfig deletion")

	var azureClusterConfig corev1alpha1.AzureClusterConfig
	{
		nsName := types.NamespacedName{
			Name:      clusterConfigName(key.ClusterID(&azureCluster)),
			Namespace: azureCluster.Namespace,
		}
		err = r.ctrlClient.Get(ctx, nsName, &azureClusterConfig)
		if errors.IsNotFound(err) {
			// Done. AzureClusterConfig is gone and finalizer can be released.
			r.logger.LogCtx(ctx, "level", "debug", "message", "AzureClusterConfig deleted")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	// Wait until AzureClusterConfig is gone.
	finalizerskeptcontext.SetKept(ctx)

	if key.IsDeleted(&azureClusterConfig) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "AzureClusterConfig deletion in progress")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	err = r.ctrlClient.Delete(ctx, &azureClusterConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureClusterConfig deletion")

	return nil
}
