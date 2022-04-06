package migration

import (
	"context"

	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/microerror"
)

func (r *Resource) deleteOldAzureMachinePool(ctx context.Context, oldAzureMachinePool *oldcapzexpv1alpha3.AzureMachinePool) error {
	// First we manually remove all finalizers, because new CR is replacing old
	// CR, so the new one will have all required finalizers, and we don't want
	// those finalizers to block the deletion of the old AzureMachinePool.
	r.logger.Debugf(ctx, "Deleting old AzureMachinePool %s/%s", oldAzureMachinePool.Namespace, oldAzureMachinePool.Name)
	r.logger.Debugf(ctx, "Removing all old AzureMachinePool %s/%s finalizers", oldAzureMachinePool.Namespace, oldAzureMachinePool.Name)
	oldAzureMachinePool.ObjectMeta.SetFinalizers([]string{})
	err := r.ctrlClient.Update(ctx, oldAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Updated old AzureMachinePool %s/%s and removed all finalizers", oldAzureMachinePool.Namespace, oldAzureMachinePool.Name)

	// Finally, delete the old AzureMachinePool
	err = r.ctrlClient.Delete(ctx, oldAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Deleted old AzureMachinePool %s/%s", oldAzureMachinePool.Namespace, oldAzureMachinePool.Name)

	return nil
}
