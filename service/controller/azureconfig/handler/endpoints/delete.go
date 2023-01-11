package endpoints

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/crud"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v7/service/controller/key"
)

func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteChange interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	endpointsToDelete, err := toEndpoints(deleteChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if endpointsToDelete != nil {
		r.logger.Debugf(ctx, "deleting Kubernetes endpoints")

		namespace := key.ClusterNamespace(cr)
		err := r.k8sClient.CoreV1().Endpoints(namespace).Delete(ctx, endpointsToDelete.Name, apismetav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "deleting Kubernetes endpoints: deleted")
	} else {
		r.logger.Debugf(ctx, "deleting Kubernetes endpoints: already deleted")
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
	currentEndpoints, err := toEndpoints(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredEndpoints, err := toEndpoints(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "finding out if the endpoints has to be deleted")

	var endpointsToDelete *corev1.Endpoints
	if currentEndpoints != nil && desiredEndpoints.Name == currentEndpoints.Name {
		endpointsToDelete = desiredEndpoints
	}

	r.logger.Debugf(ctx, "found out if the endpoints has to be deleted")

	return endpointsToDelete, nil
}
