package spark

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

// EnsureDeleted will delete the `Spark` CR that was created for this specific node pool, and the `Secret` referenced by it.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	var sparkCR *corev1alpha1.Spark
	{
		err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureMachinePool.Namespace, Name: azureMachinePool.Name}, sparkCR)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "bootstrap CR not found when trying to delete it")
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = r.ctrlClient.Delete(ctx, sparkCR)
		if errors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "bootstrap CR not found when trying to delete it")
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
