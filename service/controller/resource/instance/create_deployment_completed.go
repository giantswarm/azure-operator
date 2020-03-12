package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/to"

	"github.com/giantswarm/azure-operator/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/resource/instance/internal/state"
)

func (r *Resource) deploymentCompletedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	d, err := deploymentsClient.Get(ctx, key.ClusterID(cr), key.VmssDeploymentName)
	if IsDeploymentNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deployment not found") // nolint: errcheck
		r.logger.LogCtx(ctx, "level", "debug", "message", "waiting for creation") // nolint: errcheck
		return currentState, nil
	} else if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	s := *d.Properties.ProvisioningState
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s)) // nolint: errcheck

	if key.IsSucceededProvisioningState(s) {
		if d.Location == nil {
			d.Location = to.StringP("<unknown>")
		}

		computedDeployment, err := r.newDeployment(ctx, cr, nil, *d.Location)
		if blobclient.IsBlobNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found") // nolint: errcheck
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

			currentDeploymentTemplateChk, err := r.getResourceStatus(cr, DeploymentTemplateChecksum)
			if err != nil {
				return "", microerror.Mask(err)
			}

			currentDeploymentParametersChk, err := r.getResourceStatus(cr, DeploymentParametersChecksum)
			if err != nil {
				return "", microerror.Mask(err)
			}

			if currentDeploymentTemplateChk != desiredDeploymentTemplateChk || currentDeploymentParametersChk != desiredDeploymentParametersChk {
				r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed") // nolint: errcheck
				// As current and desired state differs, start process from the beginning.
				return DeploymentUninitialized, nil
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged") // nolint: errcheck

			return currentState, nil
		}
	} else if key.IsFinalProvisioningState(s) {
		// Deployment has failed. Restart from beginning.
		return DeploymentUninitialized, nil
	}

	r.logger.LogCtx(ctx, "level", "warning", "message", "instances reconciliation process reached unexpected state") // nolint: errcheck

	// Normally the process should never get here. In case this happens, start
	// from the beginning.
	return DeploymentUninitialized, nil
}
