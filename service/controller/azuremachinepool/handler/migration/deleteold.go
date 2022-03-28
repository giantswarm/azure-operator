package migration

import (
	"context"

	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/microerror"
)

func (r *Resource) deleteOldAzureMachinePool(ctx context.Context, oldAzureMachinePool *oldcapzexpv1alpha3.AzureMachinePool) error {
	// First we manually remove all finalizers, because new CR is replacing old
	// CR, so the new one will have all required finalizers, and we don't want
	// those finalizers to block the deletion of the old AzureMachinePool.
	oldAzureMachinePool.ObjectMeta.SetFinalizers([]string{})
	err := r.ctrlClient.Update(ctx, oldAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	// Finally, delete the old AzureMachinePool
	err = r.ctrlClient.Delete(ctx, oldAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
