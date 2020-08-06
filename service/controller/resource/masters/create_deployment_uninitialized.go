package masters

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

	"github.com/giantswarm/azure-operator/v4/pkg/checksum"
	"github.com/giantswarm/azure-operator/v4/service/controller/blobclient"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) deploymentUninitializedTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return currentState, microerror.Mask(err)
	}
	deploymentsClient, err := r.ClientFactory.GetDeploymentsClient(key.CredentialNamespace(cr), key.CredentialName(cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}
	groupsClient, err := r.ClientFactory.GetGroupsClient(key.CredentialNamespace(cr), key.CredentialName(cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}
	virtualMachineScaleSetVMsClient, err := r.ClientFactory.GetVirtualMachineScaleSetVMsClient(key.CredentialNamespace(cr), key.CredentialName(cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	group, err := groupsClient.Get(ctx, key.ClusterID(&cr))
	if err != nil {
		return currentState, microerror.Mask(err)
	}

	computedDeployment, err := r.newDeployment(ctx, cr, nil, *group.Location)
	if controllercontext.IsInvalidContext(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", err.Error())
		r.Logger.LogCtx(ctx, "level", "debug", "message", "missing dispatched output values in controller context")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "did not ensure deployment")
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
		return currentState, nil
	} else if blobclient.IsBlobNotFound(err) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
		resourcecanceledcontext.SetCanceled(ctx)
		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
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

		r.Logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment")

		deploymentTemplateChk, err := checksum.GetDeploymentTemplateChecksum(computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if deploymentTemplateChk != "" {
			err = r.SetResourceStatus(ctx, cr, DeploymentTemplateChecksum, deploymentTemplateChk)
			if err != nil {
				return currentState, microerror.Mask(err)
			}

			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentTemplateChecksum, deploymentTemplateChk))
		} else {
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentTemplateChecksum))
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

			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentParametersChecksum, deploymentParametersChk))
		} else {
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentParametersChecksum))
		}

		// Start watcher on the instances to avoid stuck VMs to block the deployment progress forever
		r.InstanceWatchdog.DeleteFailedVMSS(ctx, virtualMachineScaleSetVMsClient, key.ResourceGroupName(cr), key.MasterVMSSName(cr))

		r.Logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)

		return DeploymentInitialized, nil
	}
}
