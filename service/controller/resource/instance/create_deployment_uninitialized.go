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

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	group, err := groupsClient.Get(ctx, key.ClusterID(cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	computedDeployment, err := r.newDeployment(ctx, cr, nil, *group.Location)
	if controllercontext.IsInvalidContext(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", err.Error())
		r.logger.LogCtx(ctx, "level", "debug", "message", "missing dispatched output values in controller context")
		r.logger.LogCtx(ctx, "level", "debug", "message", "did not ensure deployment")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	} else if blobclient.IsBlobNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
		resourcecanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
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

		r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment")

		deploymentTemplateChk, err := getDeploymentTemplateChecksum(computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if deploymentTemplateChk != "" {
			err = r.setResourceStatus(cr, DeploymentTemplateChecksum, deploymentTemplateChk)
			if err != nil {
				return currentState, microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentTemplateChecksum, deploymentTemplateChk))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentTemplateChecksum))
		}

		deploymentParametersChk, err := getDeploymentParametersChecksum(computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if deploymentParametersChk != "" {
			err = r.setResourceStatus(cr, DeploymentParametersChecksum, deploymentParametersChk)
			if err != nil {
				return currentState, microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentParametersChecksum, deploymentParametersChk))
		} else {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentParametersChecksum))
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)

		return DeploymentInitialized, nil
	}
}
