package crmapper

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/finalizerskeptcontext"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	azureCluster, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring AzureConfig deletion")

	var azureConfig providerv1alpha1.AzureConfig
	{
		nsName := types.NamespacedName{
			Name:      key.ClusterID(&azureCluster),
			Namespace: metav1.NamespaceDefault,
		}
		err = r.ctrlClient.Get(ctx, nsName, &azureConfig)
		if errors.IsNotFound(err) {
			// Done. AzureConfig is gone and finalizer can be released.
			r.logger.LogCtx(ctx, "level", "debug", "message", "AzureConfig deleted")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	// Wait until AzureConfig is gone.
	finalizerskeptcontext.SetKept(ctx)

	if key.IsDeleted(&azureConfig) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "AzureConfig deletion in progress")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return nil
	}

	err = r.ctrlClient.Delete(ctx, &azureConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured AzureConfig deletion")

	return nil
}
