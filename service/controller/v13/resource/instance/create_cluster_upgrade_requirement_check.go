package instance

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v13/blobclient"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
)

func (r *Resource) clusterUpgradeRequirementCheckTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Check for changes that must not recycle the nodes but just apply the
	// VMSS deployment.
	isScaling, err := r.isClusterScaling(ctx, cr)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if isScaling {
		// When cluster is scaling we skip upgrading master node[s] and
		// re-creating worker instances.
		return DeploymentCompleted, nil
	}

	computedDeployment, err := r.newDeployment(ctx, cr, nil)
	if blobclient.IsBlobNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ignition blob not found")
		return currentState, nil
	} else if err != nil {
		return currentState, microerror.Mask(err)
	} else {
		desiredDeploymentTemplateChk, err := getDeploymentTemplateChecksum(computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		desiredDeploymentParametersChk, err := getDeploymentParametersChecksum(computedDeployment)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		currentDeploymentTemplateChk, err := r.getResourceStatus(cr, DeploymentTemplateChecksum)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		currentDeploymentParametersChk, err := r.getResourceStatus(cr, DeploymentParametersChecksum)
		if err != nil {
			return currentState, microerror.Mask(err)
		}

		if currentDeploymentTemplateChk != desiredDeploymentTemplateChk || currentDeploymentParametersChk != desiredDeploymentParametersChk {
			r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")
			// As current and desired state differs, start process from the beginning.
			return MasterInstancesUpgrading, nil
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")

		return DeploymentCompleted, nil
	}
}

func (r *Resource) isClusterScaling(ctx context.Context, cr providerv1alpha1.AzureConfig) (bool, error) {
	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}

	vmss, err := c.Get(ctx, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))
	if err != nil {
		return false, microerror.Mask(err)
	}

	isScaling := key.WorkerCount(cr) != int(*vmss.Sku.Capacity)
	return isScaling, nil
}
