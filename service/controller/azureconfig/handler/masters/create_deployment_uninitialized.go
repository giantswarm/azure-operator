package masters

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/reconciliationcanceledcontext"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/v8/pkg/checksum"
	"github.com/giantswarm/azure-operator/v8/pkg/handler/nodes/state"
	"github.com/giantswarm/azure-operator/v8/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v8/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

func (r *Resource) deploymentUninitializedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}
	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(ctx, cr.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}
	groupsClient, err := r.ClientFactory.GetGroupsClient(ctx, cr.ObjectMeta)
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.Logger.Debugf(ctx, "ensuring deployment")

	group, err := groupsClient.Get(ctx, key.ClusterID(&cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	computedDeployment, err := r.newDeployment(ctx, cr, nil, *group.Location)
	if controllercontext.IsInvalidContext(err) {
		r.Logger.Debugf(ctx, err.Error())
		r.Logger.Debugf(ctx, "missing dispatched output values in controller context")
		r.Logger.Debugf(ctx, "did not ensure deployment")
		r.Logger.Debugf(ctx, "canceling resource")
		return currentState, nil
	} else if blobclient.IsBlobNotFound(err) {
		r.Logger.Debugf(ctx, "ignition blob not found")
		resourcecanceledcontext.SetCanceled(ctx)
		r.Logger.Debugf(ctx, "canceling resource")
		return currentState, nil
	} else if err != nil {
		return currentState, microerror.Mask(err)
	} else {
		res, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(&cr), key.MastersVmssDeploymentName, computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		_, err = deploymentsClient.CreateOrUpdateResponder(res.Response())
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		r.Logger.Debugf(ctx, "ensured deployment")

		deploymentTemplateChk, err := checksum.GetDeploymentTemplateChecksum(computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if deploymentTemplateChk != "" {
			err = r.SetResourceStatus(ctx, cr, DeploymentTemplateChecksum, deploymentTemplateChk)
			if err != nil {
				return currentState, microerror.Mask(err)
			}

			r.Logger.Debugf(ctx, "set %s to '%s'", DeploymentTemplateChecksum, deploymentTemplateChk)
		} else {
			r.Logger.Debugf(ctx, "Unable to get a valid Checksum for %s", DeploymentTemplateChecksum)
		}

		deploymentParametersChk, err := checksum.GetDeploymentParametersChecksum(computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if deploymentParametersChk != "" {
			err = r.SetResourceStatus(ctx, cr, DeploymentParametersChecksum, deploymentParametersChk)
			if err != nil {
				return currentState, microerror.Mask(err)
			}

			r.Logger.Debugf(ctx, "set %s to '%s'", DeploymentParametersChecksum, deploymentParametersChk)
		} else {
			r.Logger.Debugf(ctx, "Unable to get a valid Checksum for %s", DeploymentParametersChecksum)
		}

		r.Logger.Debugf(ctx, "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)

		return DeploymentInitialized, nil
	}
}
