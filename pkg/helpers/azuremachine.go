package helpers

import (
	"context"

	"github.com/giantswarm/apiextensions/v5/pkg/annotation"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func InitAzureMachineAnnotations(ctx context.Context, ctrlClient client.Client, logger micrologger.Logger, azureMachine *capz.AzureMachine) error {
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

	logger.Debugf(ctx, "ensuring that azuremachine has release.giantswarm.io/last-deployed-version annotation initialized")

	// Initialize annotation only for the CRs that do not have it set already.
	// This will ensure that AzureMachine Creating and Upgrading conditions are
	// set properly for the first time.

	clusterName := key.ClusterName(azureMachine)
	cluster, err := util.GetClusterByName(ctx, ctrlClient, azureMachine.Namespace, clusterName)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterLastDeployedReleaseVersion, ok := cluster.Annotations[annotation.LastDeployedReleaseVersion]
	if !ok {
		// Cluster CR does not have release.giantswarm.io/last-deployed-version
		// annotation set, which means that the Cluster is probably being
		// created, so we don't have to initialize that same annotation for
		// AzureMachine CR, as it will be updated by Cluster controller
		// clusterupgrade handler.
		return nil
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
	azureMachine.Annotations[annotation.LastDeployedReleaseVersion] = clusterLastDeployedReleaseVersion

	return nil
}
