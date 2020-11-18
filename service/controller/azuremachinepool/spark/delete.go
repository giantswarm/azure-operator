package spark

import (
	"context"
	"fmt"

	corev1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// EnsureDeleted will delete the `Spark` CR that was created for this specific node pool, and the `Secret` referenced by it.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.deleteBootstrapCR(ctx, azureMachinePool.Namespace, azureMachinePool.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.deleteBootstrapSecret(ctx, azureMachinePool.Namespace, azureMachinePool.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) deleteBootstrapCR(ctx context.Context, namespace, bootstrapCRName string) error {
	var deletePropagationForeground = &client.DeleteAllOfOptions{
		DeleteOptions: client.DeleteOptions{
			PropagationPolicy: toDeletePropagationP(metav1.DeletePropagationForeground),
		},
	}

	sparkCR := &corev1alpha1.Spark{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      bootstrapCRName,
		},
		Spec: corev1alpha1.SparkSpec{},
	}
	err := r.ctrlClient.Delete(ctx, sparkCR, deletePropagationForeground)
	if errors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "bootstrap CR not found when trying to delete it")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("bootstrap CR %#q deleted", bootstrapCRName))

	return nil
}

func (r *Resource) deleteBootstrapSecret(ctx context.Context, namespace, bootstrapCRName string) error {
	err := r.ctrlClient.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName(bootstrapCRName),
		},
	})
	if errors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "bootstrap Secret not found when trying to delete it")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("bootstrap Secret %#q deleted", secretName(bootstrapCRName)))

	return nil
}

func toDeletePropagationP(v metav1.DeletionPropagation) *metav1.DeletionPropagation {
	return &v
}
