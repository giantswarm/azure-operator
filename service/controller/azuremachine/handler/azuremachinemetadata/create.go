package azuremachinemetadata

import (
	"context"

	"github.com/giantswarm/apiextensions/v6/pkg/annotation"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/azure-operator/v7/pkg/helpers"
	"github.com/giantswarm/azure-operator/v7/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, cr interface{}) error {
	var err error
	azureMachine, err := key.ToAzureMachine(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	changed := false
	_, annotationWasSet := azureMachine.Annotations[annotation.LastDeployedReleaseVersion]

	// We need this for existing clusters, and we do it both in Cluster
	// controller (clusterupgrade) and in AzureMachine controller, because we
	// do not know which controller/handler will be executed first, and we need
	// annotation set correctly in both places.
	err = helpers.InitAzureMachineAnnotations(ctx, r.ctrlClient, r.logger, &azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	_, annotationIsSet := azureMachine.Annotations[annotation.LastDeployedReleaseVersion]
	changed = !annotationWasSet && annotationIsSet

	if changed {
		err = r.ctrlClient.Update(ctx, &azureMachine)
		if errors.IsConflict(err) {
			r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
			r.logger.Debugf(ctx, "cancelling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	r.logger.Debugf(ctx, "ensured that azuremachine has release.giantswarm.io/last-deployed-version annotation initialized")

	return nil
}
