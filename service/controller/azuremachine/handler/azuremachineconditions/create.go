package azuremachineconditions

import (
	"context"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

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

	// ensure Creating condition
	err = r.creatingConditionHandler.EnsureCreated(ctx, &azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	// ensure Upgrading condition
	err = r.upgradingConditionHandler.EnsureCreated(ctx, &azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Status().Update(ctx, &azureMachine)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
