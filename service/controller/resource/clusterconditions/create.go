package clusterconditions

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, cr interface{}) error {
	var err error
	cluster, err := key.ToCluster(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	// ensure condition Ready
	err = r.ensureReadyCondition(ctx, &cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	// ensure condition Creating
	err = r.ensureCreatingCondition(ctx, &cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	// ensure condition Upgrading
	err = r.ensureUpgradingCondition(ctx, &cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ctrlClient.Status().Update(ctx, &cluster)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
