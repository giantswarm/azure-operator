package masters

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/to"

	"github.com/giantswarm/azure-operator/v4/pkg/checksum"
	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) deploymentCompletedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return Empty, microerror.Mask(err)
	}
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return Empty, microerror.Mask(err)
	}

	d, err := deploymentsClient.Get(ctx, key.ClusterID(cr), key.MastersVmssDeploymentName)
	if IsDeploymentNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "deployment should be completed but is not found")
		r.logger.LogCtx(ctx, "level", "debug", "message", "going back to DeploymentUninitialized")
		return Empty, nil
	} else if err != nil {
		return Empty, microerror.Mask(err)
	}

	s := *d.Properties.ProvisioningState
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

	if key.IsSucceededProvisioningState(s) {
		if d.Location == nil {
			d.Location = to.StringP("<unknown>")
		}

		computedDeployment, err := r.newDeployment(ctx, cr, nil, *d.Location)
		if blobclient.IsBlobNotFound(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		} else {
			desiredDeploymentTemplateChk, err := checksum.GetDeploymentTemplateChecksum(computedDeployment)
			if err != nil {
				return "", microerror.Mask(err)
			}

			desiredDeploymentParametersChk, err := checksum.GetDeploymentParametersChecksum(computedDeployment)
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
				r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")
				// As current and desired state differs, start process from the beginning.
				return Empty, nil
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")

			return currentState, nil
		}
	} else if key.IsFinalProvisioningState(s) {
		// Deployment has failed. Restart from beginning.
		return Empty, nil
	}

	r.logger.LogCtx(ctx, "level", "warning", "message", "instances reconciliation process reached unexpected state")

	// Normally the process should never get here. In case this happens, start
	// from the beginning.
	return Empty, nil
}
