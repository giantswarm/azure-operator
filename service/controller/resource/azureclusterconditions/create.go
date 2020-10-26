package azureclusterconditions

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, cr interface{}) error {
	var err error
	azureCluster, err := key.ToAzureCluster(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	// ensure Ready condition
	r.ensureReadyCondition(ctx, &azureCluster)

	err = r.ctrlClient.Status().Update(ctx, &azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
