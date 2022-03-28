package migration

import (
	"context"

	"github.com/giantswarm/apiextensions/v5/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/microerror"
)

func (r *Resource) deleteOldMachinePool(ctx context.Context, oldMachinePool *v1alpha3.MachinePool) error {
	// First we manually remove all finalizers, because new CR is replacing old
	// CR, so the new one will have all required finalizers, and we don't want
	// those finalizers to block the deletion of the old MachinePool.
	oldMachinePool.ObjectMeta.SetFinalizers([]string{})
	err := r.ctrlClient.Update(ctx, oldMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	// Finally, delete the old MachinePool
	err = r.ctrlClient.Delete(ctx, oldMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
