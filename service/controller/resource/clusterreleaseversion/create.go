package clusterreleaseversion

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

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

	if capiconditions.IsTrue(&cluster, aeconditions.CreatingCondition) {
		updateReleaseVersion, err = r.isCreationCompleted(ctx, &cluster)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if capiconditions.IsTrue(&cluster, aeconditions.UpgradingCondition) {
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
