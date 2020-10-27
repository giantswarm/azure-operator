package machinepoolconditions

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, cr interface{}) error {
	var err error
	machinePool, err := key.ToMachinePool(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	// ensure Ready condition
	err = r.ensureReadyCondition(ctx, &machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Status().Update(ctx, &machinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
