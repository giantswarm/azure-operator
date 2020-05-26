package masters

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

// This transition function aims at detecting if the master VMSS needs to be migrated from CoreOS to flatcar.
func (r *Resource) checkFlatcarMigrationNeededTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	legacyExists, err := r.vmssExistsAndHasActiveInstance(ctx, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	if !legacyExists {
		// The legacy VMSS does not exist, we can't migrate anything so we skip the migration states and go straight
		// to the standard reconciliation state.
		return DeploymentUninitialized, nil
	}

	newExists, err := r.vmssExistsAndHasActiveInstance(ctx, key.ResourceGroupName(cr), key.MasterVMSSName(cr))
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	if newExists {
		// We have both a running legacy master and a running new master.
		// Manual intervention is required in order to fix the situation.
		r.Logger.LogCtx(ctx, "level", "error", "message", "Both an old and a new master VMSS are running. This is critital and must be handled manually.")
		return ManualInterventionRequired, nil
	}

	return WaitForBackupConfirmation, nil
}

func (r *Resource) vmssExistsAndHasActiveInstance(ctx context.Context, resourceGroup string, vmssName string) (bool, error) {
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Checking if the VMSS %s exists in resource group %s", vmssName, resourceGroup)) // nolint: errcheck

	runningInstances, err := r.getRunningInstances(ctx, resourceGroup, vmssName)
	if IsScaleSetNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return len(runningInstances) > 0, nil
}
