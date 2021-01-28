package azuremachinemetadata

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, cr interface{}) error {
	var err error
	azureMachine, err := key.ToAzureMachine(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureAnnotations(ctx, &azureMachine)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureAnnotations(ctx context.Context, azureMachine *capz.AzureMachine) error {
	// We want to initialize release.giantswarm.io/last-deployed-version annotation
	// and set it to the whatever is the latest deployed release version.
	// This will ensure that AzureMachine CRs for existing clusters get Creating
	// and Upgrading conditions correctly initialized.
	if azureMachine.Annotations == nil {
		azureMachine.Annotations = map[string]string{}
	}

	_, ok := azureMachine.Annotations[annotation.LastDeployedReleaseVersion]
	if ok {
		return nil
	}

	r.logger.Debugf(ctx, "ensuring that azuremachine has release.giantswarm.io/last-deployed-version annotation initialized")

	// Initialize annotation only for the CRs that do not have it set already.
	// This will ensure that AzureMachine Creating and Upgrading conditions are
	// set properly for the first time.

	clusterName := key.ClusterName(azureMachine)
	cluster, err := util.GetClusterByName(ctx, r.ctrlClient, azureMachine.Namespace, clusterName)
	if err != nil {
		return microerror.Mask(err)
	}

	// Since this will be executed only once (during the first upgrade to the
	// release that includes this change), we will ready the annotation value
	// from the Cluster CR, as that will be the latest deployed release at that
	// moment.
	// Few more details:
	// - Cluster release label will store newer desired release, so we
	//   cannot use that.
	// - Also ,we cannot know if AzureMachine release label was already updated
	//   or not, as that is done in Cluster controller.
	azureMachine.Annotations[annotation.LastDeployedReleaseVersion] = cluster.Annotations[annotation.LastDeployedReleaseVersion]

	err = r.ctrlClient.Update(ctx, azureMachine)
	if apierrors.IsConflict(err) {
		r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
		r.logger.Debugf(ctx, "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "ensured that azuremachine has release.giantswarm.io/last-deployed-version annotation initialized")
	return nil
}
