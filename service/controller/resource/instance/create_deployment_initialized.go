package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) deploymentInitializedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	d, err := deploymentsClient.Get(ctx, key.ClusterID(cr), key.WorkersVmssDeploymentName)
	if IsDeploymentNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deployment not found")
		r.logger.LogCtx(ctx, "level", "debug", "message", "waiting for creation")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	} else if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	s := *d.Properties.ProvisioningState
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

	if !key.IsSucceededProvisioningState(s) {
		r.debugger.LogFailedDeployment(ctx, d, err)

		if key.IsFinalProvisioningState(s) {
			// Deployment is not running and not succeeded (Failed?)
			// This indicates some kind of error in the deployment template and/or parameters.
			// Restart state machine on the next loop to apply the deployment once again.
			// (If the azure operator has been fixed/updated in the meantime that could lead to a fix).
			return DeploymentUninitialized, nil
		} else {
			return currentState, nil
		}
	}

	return ProvisioningSuccessful, nil
}
