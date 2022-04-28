package migration

import (
	"context"

	oldcapiexp "github.com/giantswarm/apiextensions/v6/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/microerror"
)

const (
	operatorkitMachinePoolExpFinalizer = "operatorkit.giantswarm.io/azure-operator-machine-pool-exp-controller"
)

func (r *Resource) deleteOldMachinePool(ctx context.Context, oldMachinePool *oldcapiexp.MachinePool) error {
	r.logger.Debugf(ctx, "Deleting old MachinePool %s/%s", oldMachinePool.Namespace, oldMachinePool.Name)
	var err error
	finalizersUpdated := false
	r.logger.Debugf(ctx, "Checking if old MachinePool %s/%s finalizers should be removed", oldMachinePool.Namespace, oldMachinePool.Name)

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
				finalizersUpdated = true
			}
		} else {
			oldMachinePool.ObjectMeta.SetFinalizers([]string{})
			err = r.ctrlClient.Update(ctx, oldMachinePool)
			finalizersUpdated = true
		}
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if finalizersUpdated {
		r.logger.Debugf(ctx, "Removed old MachinePool %s/%s finalizers", oldMachinePool.Namespace, oldMachinePool.Name)
	} else {
		r.logger.Debugf(ctx, "No need to remove old MachinePool %s/%s finalizers", oldMachinePool.Namespace, oldMachinePool.Name)
	}

	// Finally, delete the old MachinePool
	err = r.ctrlClient.Delete(ctx, oldMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Deleted old MachinePool %s/%s", oldMachinePool.Namespace, oldMachinePool.Name)

	return nil
}
