package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/azure-operator/service/controller/v14/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/v14/key"
	"github.com/giantswarm/azure-operator/service/controller/v14/resource/instance/internal/state"
	"github.com/giantswarm/microerror"
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
		computedDeployment, err := r.newDeployment(ctx, customObject, nil)
		if blobclient.IsBlobNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		} else {
			desiredDeploymentTemplateChk, err := getDeploymentTemplateChecksum(computedDeployment)
			if err != nil {
				return "", microerror.Mask(err)
			}

			desiredDeploymentParametersChk, err := getDeploymentParametersChecksum(computedDeployment)
			if err != nil {
				return "", microerror.Mask(err)
			}

			currentDeploymentTemplateChk, err := r.getResourceStatus(customObject, DeploymentTemplateChecksum)
			if err != nil {
				return "", microerror.Mask(err)
			}

			currentDeploymentParametersChk, err := r.getResourceStatus(customObject, DeploymentParametersChecksum)
			if err != nil {
				return "", microerror.Mask(err)
			}

			if currentDeploymentTemplateChk != desiredDeploymentTemplateChk || currentDeploymentParametersChk != desiredDeploymentParametersChk {
				r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")
				// As current and desired state differs, start process from the beginning.
				return DeploymentUninitialized, nil
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")

			return currentState, nil
		}
	} else if key.IsFinalProvisioningState(s) {
		// Deployment has failed. Restart from beginning.
		return DeploymentUninitialized, nil
	}

	r.logger.LogCtx(ctx, "level", "warning", "message", "instances reconciliation process reached unexpected state")

	// Normally the process should never get here. In case this happens, start
	// from the beginning.
	return DeploymentUninitialized, nil
}
