package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/pkg/checksum"
	"github.com/giantswarm/azure-operator/v5/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) deploymentCompletedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(ctx, cr.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}
	groupsClient, err := r.ClientFactory.GetGroupsClient(ctx, cr.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	d, err := deploymentsClient.Get(ctx, key.ClusterID(&cr), key.WorkersVmssDeploymentName)
	if IsDeploymentNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "deployment should be completed but is not found")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "going back to DeploymentUninitialized")
		return DeploymentUninitialized, nil
	} else if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	s := *d.Properties.ProvisioningState
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

	group, err := groupsClient.Get(ctx, key.ClusterID(&cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	if key.IsSucceededProvisioningState(s) {
		computedDeployment, err := r.newDeployment(ctx, cr, nil, *group.Location)
		if blobclient.IsBlobNotFound(err) {
			r.Logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
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

			currentDeploymentTemplateChk, err := r.GetResourceStatus(ctx, cr, DeploymentTemplateChecksum)
			if err != nil {
				return "", microerror.Mask(err)
			}

			currentDeploymentParametersChk, err := r.GetResourceStatus(ctx, cr, DeploymentParametersChecksum)
			if err != nil {
				return "", microerror.Mask(err)
			}

			if currentDeploymentTemplateChk != desiredDeploymentTemplateChk || currentDeploymentParametersChk != desiredDeploymentParametersChk {
				r.Logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")
				// As current and desired state differs, start process from the beginning.
				return DeploymentUninitialized, nil
			}

			r.Logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")

			return currentState, nil
		}
	} else if key.IsFinalProvisioningState(s) {
		// Deployment has failed. Restart from beginning.
		return DeploymentUninitialized, nil
	}

	r.Logger.LogCtx(ctx, "level", "warning", "message", "instances reconciliation process reached unexpected state")

	// Normally the process should never get here. In case this happens, start
	// from the beginning.
	return DeploymentUninitialized, nil
}
