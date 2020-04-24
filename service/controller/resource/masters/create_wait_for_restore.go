package masters

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) waitForRestoreTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	// Check if the Legacy master VMSS exists
	legacyExists, err := r.vmssExists(ctx, key.ResourceGroupName(cr), key.LegacyMasterVMSSName(cr))
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	if !legacyExists {
		// The legacy VMSS does not exist, we assume there is no need for restoring a backup.
		return DeploymentCompleted, nil
	}

	r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("The reconciliation on the masters resource is stopped until the ETCD backup is restored. When you completed the restore, set the masters's resource status to '%s'", DeleteLegacyVMSS))
	return currentState, nil
}

func (r *Resource) vmssExists(ctx context.Context, resourceGroup string, vmssName string) (bool, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Checking if the VMSS %s exists in resource group %s", vmssName, resourceGroup)) // nolint: errcheck

	_, err := r.getVMSS(ctx, resourceGroup, vmssName)
	if IsScaleSetNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}
