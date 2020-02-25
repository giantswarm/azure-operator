package instance

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/key"
	state2 "github.com/giantswarm/azure-operator/service/controller/resource/instance/internal/state"
)

func (r *Resource) clusterUpgradeRequirementCheckTransition(ctx context.Context, obj interface{}, currentState state2.State) (state2.State, error) {
	cr, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Check for changes that must not recycle the nodes but just apply the
	// VMSS deployment.
	isCreating := r.isClusterCreating(cr)
	isScaling, err := r.isClusterScaling(ctx, cr)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if isCreating || isScaling {
		// When cluster is creating or scaling we skip upgrading master node[s]
		// and replacing worker instances.
		return DeploymentCompleted, nil
	}

	return MasterInstancesUpgrading, nil
}

func (r *Resource) isClusterCreating(cr providerv1alpha1.AzureConfig) bool {
	// When cluster creation is in the beginning, it doesn't necessarily have
	// any status conditions yet.
	if len(cr.Status.Cluster.Conditions) == 0 {
		return true
	}
	if cr.Status.Cluster.HasCreatingCondition() {
		return true
	}

	return false
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
