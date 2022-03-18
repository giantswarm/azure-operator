package service

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/resource/crud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateChange interface{}) error {
	serviceToUpdate, err := toService(updateChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if serviceToUpdate != nil && serviceToUpdate.Spec.ClusterIP != "" {
		r.logger.Debugf(ctx, "updating services")

		_, err := r.k8sClient.CoreV1().Services(serviceToUpdate.Namespace).Update(ctx, serviceToUpdate, metav1.UpdateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "updated services")
	} else {
		r.logger.Debugf(ctx, "no need to update services")
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

// Service resources are updated.
func (r *Resource) newUpdateChange(ctx context.Context, currentState, desiredState interface{}) (interface{}, error) {
	currentService, err := toService(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredService, err := toService(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "finding out which services have to be updated")

	if isServiceModified(desiredService, currentService) {
		// Make a copy and set the resource version so the service can be updated.
		serviceToUpdate := desiredService.DeepCopy()
		if currentService != nil {
			serviceToUpdate.ObjectMeta.ResourceVersion = currentService.ObjectMeta.ResourceVersion
			serviceToUpdate.Spec.ClusterIP = currentService.Spec.ClusterIP
		}
		r.logger.Debugf(ctx, "found service '%s' that has to be updated", desiredService.GetName())

		return serviceToUpdate, nil
	} else {
		r.logger.Debugf(ctx, "no services needs update")

		return nil, nil
	}
}
