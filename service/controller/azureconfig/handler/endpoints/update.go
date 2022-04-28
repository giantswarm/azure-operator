package endpoints

import (
	"context"
	"reflect"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/crud"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	endpointsToUpdate, err := toEndpoints(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if endpointsToUpdate != nil {
		r.logger.Debugf(ctx, "updating Kubernetes endpoints")

		namespace := key.ClusterNamespace(cr)
		_, err := r.k8sClient.CoreV1().Endpoints(namespace).Update(ctx, endpointsToUpdate, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated Kubernetes endpoints")

	} else {
		r.logger.Debugf(ctx, "Kubernetes endpoints do not need to be updated")
	}

	return nil
}

func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	create, err := r.newCreateChange(currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	update, err := r.newUpdateChange(ctx, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch := crud.NewPatch()
	patch.SetCreateChange(create)
	patch.SetUpdateChange(update)

	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, currentState, desiredState interface{}) (interface{}, error) {
	currentEndpoints, err := toEndpoints(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredEndpoints, err := toEndpoints(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "finding out if the endpoints has to be updated")

	var endpointsToUpdate *corev1.Endpoints

	// The subsets can change if the private IP of the master node has changed.
	// We then need to update the endpoints resource.
	if currentEndpoints != nil && desiredEndpoints != nil {
		if !reflect.DeepEqual(desiredEndpoints.Subsets, currentEndpoints.Subsets) {
			endpointsToUpdate = desiredEndpoints
		}
	}

	r.logger.Debugf(ctx, "found out if the endpoints has to be deleted")

	return endpointsToUpdate, nil
}
