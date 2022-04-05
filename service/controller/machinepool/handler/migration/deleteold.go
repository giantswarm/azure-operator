package migration

import (
	"context"

	"github.com/giantswarm/apiextensions/v6/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/microerror"
)

const (
	operatorkitMachinePoolExpFinalizer = "operatorkit.giantswarm.io/azure-operator-machine-pool-exp-controller"
)

func (r *Resource) deleteOldMachinePool(ctx context.Context, oldMachinePool *v1alpha3.MachinePool) error {
	var err error
	if len(oldMachinePool.ObjectMeta.Finalizers) > 0 {
		// First we manually remove all finalizers (except for operatorkit finalizer
		// for exp MachinePool which will be removed by the operatorkit). We do
		// this because new CR is replacing old CR, therefore all operators will
		// now use the new CR and ignore the old one. We don't want that
		// finalizers to block the deletion of the old MachinePool, so we remove
		// them manually.
		operatorkitFinalizerFound := false
		for _, s := range oldMachinePool.ObjectMeta.Finalizers {
			if s == operatorkitMachinePoolExpFinalizer {
				operatorkitFinalizerFound = true
				break
			}
		}

		if operatorkitFinalizerFound {
			// Just keep the operatorkit finalizer
			if len(oldMachinePool.ObjectMeta.Finalizers) > 1 {
				// update only if we really have more than one we already want
				oldMachinePool.ObjectMeta.SetFinalizers([]string{operatorkitMachinePoolExpFinalizer})
				err = r.ctrlClient.Update(ctx, oldMachinePool)
			}
		} else {
			oldMachinePool.ObjectMeta.SetFinalizers([]string{})
			err = r.ctrlClient.Update(ctx, oldMachinePool)
		}
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Finally, delete the old MachinePool
	err = r.ctrlClient.Delete(ctx, oldMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
