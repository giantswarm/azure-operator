package namespace

import (
	"context"

	"github.com/giantswarm/microerror"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	namespaceToCreate, err := toNamespace(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if namespaceToCreate != nil {
		r.logger.Debugf(ctx, "creating Kubernetes namespace")

		_, err = r.k8sClient.CoreV1().Namespaces().Create(ctx, namespaceToCreate, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "creating Kubernetes namespace: created")
	} else {
		r.logger.Debugf(ctx, "creating Kubernetes namespace: already created")
	}

	return nil
}

func (r *Resource) newCreateChange(currentState, desiredState interface{}) (interface{}, error) {
	currentNamespace, err := toNamespace(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredNamespace, err := toNamespace(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var namespaceToCreate *corev1.Namespace
	if currentNamespace == nil {
		namespaceToCreate = desiredNamespace
	}

	return namespaceToCreate, nil
}
