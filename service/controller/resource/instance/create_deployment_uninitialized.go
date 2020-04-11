package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/resource/instance/internal/state"
)

func (r *Resource) deploymentUninitializedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return currentState, microerror.Mask(err)
	}
	groupsClient, err := r.getGroupsClient(ctx)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment") // nolint: errcheck

	group, err := groupsClient.Get(ctx, key.ClusterID(cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	computedDeployment, err := r.newDeployment(ctx, cr, nil, *group.Location)
	if controllercontext.IsInvalidContext(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", err.Error())                                              // nolint: errcheck
		r.logger.LogCtx(ctx, "level", "debug", "message", "missing dispatched output values in controller context") // nolint: errcheck
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not ensure deployment")                              // nolint: errcheck
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")                                     // nolint: errcheck
		return currentState, nil
	} else if blobclient.IsBlobNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found") // nolint: errcheck
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource") // nolint: errcheck
		return currentState, nil
	} else if err != nil {
		return currentState, microerror.Mask(err)
	} else {
		res, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(cr), key.VmssDeploymentName, computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		_, err = deploymentsClient.CreateOrUpdateResponder(res.Response())
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment") // nolint: errcheck

		deploymentTemplateChk, err := getDeploymentTemplateChecksum(computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if deploymentTemplateChk != "" {
			err = r.setResourceStatus(cr, DeploymentTemplateChecksum, deploymentTemplateChk)
			if err != nil {
				return currentState, microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentTemplateChecksum, deploymentTemplateChk)) // nolint: errcheck
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentTemplateChecksum)) // nolint: errcheck
		}

		deploymentParametersChk, err := getDeploymentParametersChecksum(computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if deploymentTemplateChk != "" {
			err = r.setResourceStatus(cr, DeploymentParametersChecksum, deploymentParametersChk)
			if err != nil {
				return currentState, microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentParametersChecksum, deploymentParametersChk)) // nolint: errcheck
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentParametersChecksum)) // nolint: errcheck
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation") // nolint: errcheck
		reconciliationcanceledcontext.SetCanceled(ctx)

		return DeploymentInitialized, nil
	}
}
