package clusterreleaseversion

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/conditions/pkg/conditions"
	"github.com/giantswarm/microerror"

	azopannotation "github.com/giantswarm/azure-operator/v5/pkg/annotation"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, cr interface{}) error {
	var err error

	cluster, err := key.ToCluster(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	updateReleaseVersion := false

	if conditions.IsCreatingTrue(&cluster) {
		updateReleaseVersion, err = r.isCreationCompleted(ctx, &cluster)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if conditions.IsUpgradingTrue(&cluster) {
		updateReleaseVersion, err = r.isUpgradeCompleted(ctx, &cluster)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	if updateReleaseVersion {
		cluster.Annotations[annotation.LastDeployedReleaseVersion] = key.ReleaseVersion(&cluster)
		if _, isUpgradingToNodePoolsSet := cluster.GetAnnotations()[azopannotation.UpgradingToNodePools]; isUpgradingToNodePoolsSet {
			delete(cluster.Annotations, azopannotation.UpgradingToNodePools)
		}

		err = r.ctrlClient.Update(ctx, &cluster)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
