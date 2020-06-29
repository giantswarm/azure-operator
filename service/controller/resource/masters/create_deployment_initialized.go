package masters

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) deploymentInitializedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}
	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(key.CredentialNamespace(cr), key.CredentialName(cr))
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	d, err := deploymentsClient.Get(ctx, key.ClusterID(&cr), key.MastersVmssDeploymentName)
	if IsDeploymentNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "deployment not found")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "waiting for creation")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	} else if err != nil {
		return Empty, microerror.Mask(err)
	}

	s := *d.Properties.ProvisioningState
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

	if !key.IsSucceededProvisioningState(s) {
		r.Debugger.LogFailedDeployment(ctx, d, err)

		if key.IsFinalProvisioningState(s) {
			// Deployment is not running and not succeeded (Failed?)
			// This indicates some kind of error in the deployment template and/or parameters.
			// Restart state machine on the next loop to apply the deployment once again.
			// (If the azure operator has been fixed/updated in the meantime that could lead to a fix).
			return Empty, nil
		} else {
			return currentState, nil
		}
	}

	return ProvisioningSuccessful, nil
}
