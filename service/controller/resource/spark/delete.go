package spark

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

// EnsureDeleted will delete the `Spark` CR that was created for this specific node pool, and the `Secret` referenced by it.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// With use of OwnerReference this deletion will also cascade to data secret object.
	err = r.ctrlClient.Delete(ctx, &corev1alpha1.Spark{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: azureMachinePool.Namespace,
			Name:      azureMachinePool.Name,
		},
		Spec: corev1alpha1.SparkSpec{},
	})
	if errors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "bootstrap CR not found when trying to delete it")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("bootstrap CR %#q deleted", azureMachinePool.Name))

	return nil
}
