package namespace

import (
	"context"

	corev2 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav2 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the namespace in the Kubernetes API")

	var namespace *corev2.Namespace
	{
		manifest, err := r.k8sClient.CoreV1().Namespaces().Get(key.ClusterNamespace(customObject), apismetav2.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the namespace in the Kubernetes API")
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "found the namespace in the Kubernetes API")
			namespace = manifest
		}
	}

	if namespace != nil && namespace.Status.Phase == "Terminating" {
		r.logger.LogCtx(ctx, "level", "debug", "message", "namespace is in state 'Terminating'")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource reconciliation for custom object")

		return nil, nil
	}

	return namespace, nil
}
