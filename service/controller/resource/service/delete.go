package service

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/resource/crud"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	serviceToDelete, err := toService(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if serviceToDelete != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting Kubernetes service")

		namespace := key.ClusterNamespace(cr)
		err := r.k8sClient.CoreV1().Services(namespace).Delete(ctx, serviceToDelete.Name, apismetav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting Kubernetes service: deleted")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deleting Kubernetes service: already deleted")
	}

	return nil
}

func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	delete, err := r.newDeleteChange(ctx, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
	patch.SetDeleteChange(delete)

	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, currentState, desiredState interface{}) (interface{}, error) {
	currentService, err := toService(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredService, err := toService(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "finding out if the service has to be deleted")

	var serviceToDelete *corev1.Service
	if currentService != nil && desiredService.Name == currentService.Name {
		serviceToDelete = desiredService
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "found out if the service has to be deleted")

	return serviceToDelete, nil
}
