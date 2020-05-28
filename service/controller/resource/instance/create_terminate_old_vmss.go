package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	legacyVMSSDeploymentName = "worker-vmss-deploy"
)

func (r *Resource) terminateOldVmssTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	c, err := r.GetScaleSetsClient(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting the legacy VMSS %s", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck
	_, err = c.Delete(ctx, key.ResourceGroupName(cr), key.LegacyWorkerVMSSName(cr))
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted the legacy VMSS %s", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting the legacy VMSS deployment %s", legacyVMSSDeploymentName)) // nolint: errcheck

	dc, err := r.ClientFactory.GetDeploymentsClient(cr)
	if err != nil {
		return "", microerror.Mask(err)
	}

	_, err = dc.Delete(ctx, key.ResourceGroupName(cr), legacyVMSSDeploymentName)
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted the legacy VMSS deployment %s", legacyVMSSDeploymentName)) // nolint: errcheck

	return DeploymentCompleted, nil
}
