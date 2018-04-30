package endpoints

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Deleting the K8s namespace will take care of cleaning the endpoints.
	if key.IsDeleted(customObject) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling endpoints deletion: deleted with the namespace")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the master endpoints in the Kubernetes API")

	namespace := key.ClusterNamespace(customObject)

	// Lookup the current state of the endpoints.
	var endpoints *corev1.Endpoints
	{
		manifest, err := r.k8sClient.CoreV1().Endpoints(namespace).Get(masterEndpointsName, apismetav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the master endpoints in the Kubernetes API")
			return nil, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "found the master endpoints in the Kubernetes API")
			endpoints = manifest
		}
	}

	return endpoints, nil
}
