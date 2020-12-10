package azuremachineconditions

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, cr interface{}) error {
	var err error
	azureMachine, err := key.ToAzureMachine(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	// ensure Ready condition
	err = r.ensureReadyCondition(ctx, &azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Status().Update(ctx, &azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
