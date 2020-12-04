package machinepoolupgrade

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/microerror"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"

	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/pkg/upgrade"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) ensureLastDeployedReleaseVersion(ctx context.Context, machinePool *capiexp.MachinePool) error {
	r.logger.Debugf(ctx, "ensuring release.giantswarm.io/last-deployed-version on MachinePool CR")
	desiredReleaseVersion := key.ReleaseVersion(machinePool)
	lastDeployedReleaseVersion := machinePool.Annotations[annotation.LastDeployedReleaseVersion]

	// release.giantswarm.io/last-deployed-version annotation is up-to-date.
	if lastDeployedReleaseVersion == desiredReleaseVersion {
		logMessage := fmt.Sprintf(
			"ensured release.giantswarm.io/last-deployed-version annotation, value %s it's already up-to-date",
			lastDeployedReleaseVersion)
		r.logger.Debugf(ctx, logMessage)
		return nil
	}

	desiredAzureOperatorVersion := key.OperatorVersion(machinePool)

	cluster, err := util.GetClusterFromMetadata(ctx, r.ctrlClient, machinePool.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	tenantClusterClient, err := r.tenantClientFactory.GetClient(ctx, cluster)
	if tenantcluster.IsAPINotAvailableError(err) {
		r.logger.Debugf(ctx, "tenant API not available yet")
		r.logger.Debugf(ctx, "canceling resource")

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}
	upgradeIsCompletedForDesiredVersion, err := upgrade.IsNodePoolUpgradeCompleted(
		ctx,
		tenantClusterClient,
		machinePool,
		desiredReleaseVersion,
		desiredAzureOperatorVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	var logMessage string
	if upgradeIsCompletedForDesiredVersion {
		machinePool.Annotations[annotation.LastDeployedReleaseVersion] = key.ReleaseVersion(machinePool)
		err = r.ctrlClient.Update(ctx, machinePool)
		if err != nil {
			return microerror.Mask(err)
		}

		logMessage = fmt.Sprintf(
			"ensured release.giantswarm.io/last-deployed-version annotation, updated to %s",
			machinePool.Annotations[annotation.LastDeployedReleaseVersion])
	} else {
		logMessage = fmt.Sprintf(
			"ensured release.giantswarm.io/last-deployed-version annotation, upgrade is not completed, value is still %s",
			machinePool.Annotations[annotation.LastDeployedReleaseVersion])
	}

	r.logger.Debugf(ctx, logMessage)
	return nil
}
