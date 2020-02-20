package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/azure-operator/service/controller/v14/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/v14/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v14/key"
	"github.com/giantswarm/azure-operator/service/controller/v14/resource/instance/internal/state"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"
)

func (r *Resource) deploymentUninitializedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	computedDeployment, err := r.newDeployment(ctx, customObject, nil)
	if controllercontext.IsInvalidContext(err) {
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
		res, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(customObject), key.VmssDeploymentName, computedDeployment)
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
			err = r.setResourceStatus(customObject, DeploymentTemplateChecksum, deploymentTemplateChk)
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

		if deploymentTemplateChk != "" {
			err = r.setResourceStatus(customObject, DeploymentParametersChecksum, deploymentParametersChk)
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
