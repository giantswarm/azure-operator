package clusterownerreference

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/reference"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) ensureControlPlane(ctx context.Context, cluster *capi.Cluster) error {
	var err error

	azureMachine := capz.AzureMachine{}
	err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cluster.Namespace, Name: key.AzureMachineName(cluster)}, &azureMachine)
	if apierrors.IsNotFound(err) {
		// Waiting for AzureMachine to be created.
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if !azureMachine.GetDeletionTimestamp().IsZero() {
		return microerror.Mask(crBeingDeletedError)
	}

	err = r.updateControlPlaneObject(ctx, cluster, &azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.updateControlPlaneRef(ctx, cluster, &azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) updateControlPlaneObject(ctx context.Context, cluster *capi.Cluster, azureMachine *capz.AzureMachine) error {
	r.logger.Debugf(ctx, "ensuring %s label and 'ownerReference' fields on AzureMachine '%s/%s'", capi.ClusterLabelName, azureMachine.Namespace, azureMachine.Name)

	// Set Cluster as owner of AzureMachine
	err := controllerutil.SetControllerReference(cluster, azureMachine, r.scheme)
	if err != nil {
		return microerror.Mask(err)
	}

	if azureMachine.Labels == nil {
		azureMachine.Labels = make(map[string]string)
	}
	azureMachine.Labels[capi.ClusterLabelName] = cluster.Name

	err = r.ctrlClient.Update(ctx, azureMachine)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured %s label and 'ownerReference' fields on AzureMachine '%s/%s'", capi.ClusterLabelName, azureMachine.Namespace, azureMachine.Name)
	return nil
}

func (r *Resource) updateControlPlaneRef(ctx context.Context, cluster *capi.Cluster, azureMachine *capz.AzureMachine) error {
	if cluster.Spec.ControlPlaneRef != nil {
		return nil
	}

	var err error
	var controlPlaneRef *corev1.ObjectReference
	{
		controlPlaneRef, err = reference.GetReference(r.scheme, azureMachine)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	cluster.Spec.ControlPlaneRef = controlPlaneRef
	err = r.ctrlClient.Update(ctx, cluster)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "canceling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured 'Spec.ControlPlaneRef' fields on Cluster '%s/%s'", cluster.Namespace, cluster.Name)
	return nil
}
