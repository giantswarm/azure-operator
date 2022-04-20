package migration

import (
	"context"

	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
)

func (r *Resource) deleteOldAzureMachinePool(ctx context.Context, oldAzureMachinePool *oldcapzexpv1alpha3.AzureMachinePool, newAzureMachinePool *capzexp.AzureMachinePool) error {
	// First we manually remove all finalizers, because new CR is replacing old
	// CR, so the new one will have all required finalizers, and we don't want
	// those finalizers to block the deletion of the old AzureMachinePool.
	update := false
	r.logger.Debugf(ctx, "Deleting old AzureMachinePool %s/%s", oldAzureMachinePool.Namespace, oldAzureMachinePool.Name)
	if len(oldAzureMachinePool.ObjectMeta.Finalizers) > 0 {
		r.logger.Debugf(ctx, "Removing all old AzureMachinePool %s/%s finalizers", oldAzureMachinePool.Namespace, oldAzureMachinePool.Name)
		oldAzureMachinePool.ObjectMeta.SetFinalizers([]string{})
		update = true
	}

	// We then update release label, in order to stop reconciling old AzureMachinePool
	// from the old azure-operator version
	if oldAzureMachinePool.Labels[label.ReleaseVersion] != newAzureMachinePool.Labels[label.ReleaseVersion] {
		r.logger.Debugf(ctx,
			"Updating release label on old AzureMachinePool %s/%s from %s to %s",
			oldAzureMachinePool.Namespace,
			oldAzureMachinePool.Name,
			oldAzureMachinePool.Labels[label.ReleaseVersion],
			newAzureMachinePool.Labels[label.ReleaseVersion])
		oldAzureMachinePool.Labels[label.ReleaseVersion] = newAzureMachinePool.Labels[label.ReleaseVersion]
		update = true
	}

	if update {
		err := r.ctrlClient.Update(ctx, oldAzureMachinePool)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Updated old AzureMachinePool %s/%s", oldAzureMachinePool.Namespace, oldAzureMachinePool.Name)
	}

	// Finally, delete the old AzureMachinePool
	err := r.ctrlClient.Delete(ctx, oldAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Deleted old AzureMachinePool %s/%s", oldAzureMachinePool.Namespace, oldAzureMachinePool.Name)

	return nil
}
