package service

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "looking for the master service in the Kubernetes API") // nolint: errcheck

	namespace := key.ClusterNamespace(cr)

	// Lookup the current state of the service.
	var service *corev1.Service
	{
		manifest, err := r.k8sClient.CoreV1().Services(namespace).Get(masterServiceName, apismetav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "did not find the master service in the Kubernetes API") // nolint: errcheck
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", "found the master service in the Kubernetes API") // nolint: errcheck
			service = manifest
		}
	}

	return service, nil
}
