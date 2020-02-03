package instance

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
	"github.com/giantswarm/microerror"
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

	return MasterInstancesUpgrading, nil
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
