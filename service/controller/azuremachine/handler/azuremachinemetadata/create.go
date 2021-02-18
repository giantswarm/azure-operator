package azuremachinemetadata

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, cr interface{}) error {
	var err error
	azureMachine, err := key.ToAzureMachine(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	// We need this for existing clusters, and we do it both in Cluster
	// controller (clusterupgrade) and in AzureMachine controller, because we
	// do not know which controller/handler will be executed first, and we need
	// annotation set correctly in both places.
	err = helpers.InitAzureMachineAnnotations(ctx, r.ctrlClient, r.logger, &azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured that azuremachine has release.giantswarm.io/last-deployed-version annotation initialized")

	return nil
}
