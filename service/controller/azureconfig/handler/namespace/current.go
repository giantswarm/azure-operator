package namespace

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v8/pkg/controller/context/finalizerskeptcontext"
	"github.com/giantswarm/operatorkit/v8/pkg/controller/context/resourcecanceledcontext"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var namespace *corev1.Namespace
	{
		r.logger.Debugf(ctx, "finding the namespace in the Kubernetes API")

		manifest, err := r.k8sClient.CoreV1().Namespaces().Get(ctx, key.ClusterNamespace(cr), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			r.logger.Debugf(ctx, "did not find the namespace in the Kubernetes API")
			// fall through
		} else if err != nil {
			return nil, microerror.Mask(err)
		} else {
			r.logger.Debugf(ctx, "found the namespace in the Kubernetes API")
			namespace = manifest
		}
	}

	// In case the namespace is already terminating we do not need to do any
	// further work. We cancel the namespace resource to prevent any further work,
	// but keep the finalizers until the namespace got finally deleted. This is to
	// prevent issues with the monitoring and alerting systems. The cluster status
	// conditions of the watched CR are used to inhibit alerts, for instance when
	// the cluster is being deleted.
	if namespace != nil && namespace.Status.Phase == corev1.NamespaceTerminating {
		r.logger.Debugf(ctx, "namespace is %#q", corev1.NamespaceTerminating)

		r.logger.Debugf(ctx, "keeping finalizers")
		finalizerskeptcontext.SetKept(ctx)

		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)

		return nil, nil
	}

	if namespace == nil && key.IsDeleted(&cr) {
		r.logger.Debugf(ctx, "resource deletion completed")

		r.logger.Debugf(ctx, "canceling resource")
		resourcecanceledcontext.SetCanceled(ctx)

		return nil, nil
	}

	return namespace, nil
}
