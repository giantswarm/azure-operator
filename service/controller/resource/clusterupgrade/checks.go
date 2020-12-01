package clusterupgrade

import (
	"github.com/giantswarm/apiextensions/v3/pkg/annotation"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/conditions/pkg/conditions"
	"github.com/giantswarm/microerror"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
)

func isAnyMachinePoolUpgrading(cr capi.Cluster, machinePools []capiexp.MachinePool) (bool, error) {
	desiredRelease := cr.Labels[label.ReleaseVersion]

	for _, machinePool := range machinePools {
		isUpgrading, err := isMachinePoolUpgradingInProgress(&machinePool, desiredRelease)
		if err != nil {
			return false, microerror.Mask(err)
		}

		if isUpgrading {
			return true, nil
		}
	}

	return false, nil
}

func isMachinePoolUpgradingInProgress(cr conditions.Object, desiredRelease string) (bool, error) {
	if conditions.IsUpgradingTrue(cr) {
		// When Upgrading == True => Upgrading is still in progress.
		return true, nil
	}

	// If release label isn't updated yet, it means that upgrade hasn't been
	// triggered yet.
	if cr.GetLabels()[label.ReleaseVersion] != desiredRelease {
		return false, nil
	}

	// release.giantswarm.io/version is updated, which means that the node pool
	// upgrade has been started, let's see if it is still in progress.

	lastDeployedReleaseVersion, isSet := cr.GetAnnotations()[annotation.LastDeployedReleaseVersion]
	if !isSet {
		// If release.giantswarm.io/last-deployed-version is not set, that
		// means that the node pool is still being created, so no upgrade in
		// progress.
		return false, nil
	}

	// If last deployed release version is equal to the desired release version
	// while Upgrading condition status is not True, it means that upgrade has
	// been completed already.
	if lastDeployedReleaseVersion == desiredRelease {
		return false, nil
	}

	// At this point we know that CR labels were updated for upgrade but the
	// condition didn't pick up yet. Therefore we consider the upgrading to be
	// in progress.
	return true, nil
}

func isMachinePoolUpgraded(cr conditions.Object, desiredRelease string) (bool, error) {
	if conditions.IsUpgradingTrue(cr) {
		// When Upgrading == True => Upgrading is still in progress.
		return false, nil
	}

	lastDeployedReleaseVersion, isSet := cr.GetAnnotations()[annotation.LastDeployedReleaseVersion]
	if !isSet {
		// if release.giantswarm.io/last-deployed-version is not set, that
		// means that the node pool is still being created, so upgrade is not
		// completed.
		return false, nil
	}

	// When CR release label matches desired version & last deployed release
	// version also matches the desired release version, we know that the
	// upgrade has completed.
	if cr.GetLabels()[label.ReleaseVersion] == desiredRelease && lastDeployedReleaseVersion == desiredRelease {
		return true, nil
	}

	return false, nil
}
