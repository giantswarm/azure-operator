package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
)

func (r *Resource) deploymentCompletedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), key.VmssDeploymentName)
	if IsDeploymentNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deployment not found")
		r.logger.LogCtx(ctx, "level", "debug", "message", "waiting for creation")
		return currentState, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	s := *d.Properties.ProvisioningState
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

	if key.IsSucceededProvisioningState(s) {
		areThereChangesToReconciliate, err := r.areThereChangesToReconciliate(err, ctx, customObject)
		if err != nil {
			return DeploymentUninitialized, microerror.Mask(err)
		}

		if areThereChangesToReconciliate {
			// As current and desired state differs, start process from the beginning.
			return DeploymentUninitialized, nil
		}

		return currentState, nil

	} else if key.IsFinalProvisioningState(s) {
		// Deployment has failed. Restart from beginning.
		return DeploymentUninitialized, nil
	}

	r.logger.LogCtx(ctx, "level", "warning", "message", "instances reconciliation process reached unexpected state")

	// Normally the process should never get here. In case this happens, start
	// from the beginning.
	return DeploymentUninitialized, nil
}
