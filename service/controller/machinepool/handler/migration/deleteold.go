package migration

import (
	"context"

	oldcapiexp "github.com/giantswarm/apiextensions/v6/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/microerror"
)

var (
	pauseAnnotations = map[string]string{
		"cluster.x-k8s.io/paused":          "true",
		"operatorkit.giantswarm.io/paused": "true",
	}
)

func (r *Resource) deleteOldMachinePool(ctx context.Context, oldMachinePool *oldcapiexp.MachinePool) error {
	r.logger.Debugf(ctx, "Deleting old MachinePool %s/%s", oldMachinePool.Namespace, oldMachinePool.Name)
	var err error

	update := false
	r.logger.Debugf(ctx, "Checking if old MachinePool %s/%s finalizers should be removed", oldMachinePool.Namespace, oldMachinePool.Name)

	// Delete all finalizers.
	if len(oldMachinePool.ObjectMeta.Finalizers) > 0 {
		oldMachinePool.ObjectMeta.SetFinalizers([]string{})
		update = true
	}

	// Ensure pause annotations are in place.
	for k, v := range pauseAnnotations {
		if oldMachinePool.GetAnnotations()[k] != v {
			if oldMachinePool.Annotations == nil {
				oldMachinePool.Annotations = make(map[string]string)
			}
			oldMachinePool.Annotations[k] = v
			update = true
		}
	}

	if update {
		err := r.ctrlClient.Update(ctx, oldMachinePool)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Removed finalizers and ensured pause annotations on old MachinePool %s/%s", oldMachinePool.Namespace, oldMachinePool.Name)
	}

	// Finally, delete the old MachinePool
	err = r.ctrlClient.Delete(ctx, oldMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Deleted old MachinePool %s/%s", oldMachinePool.Namespace, oldMachinePool.Name)

	return nil
}
