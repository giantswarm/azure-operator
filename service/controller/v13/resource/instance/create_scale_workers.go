package instance

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v13/key"
	"github.com/giantswarm/azure-operator/service/controller/v13/resource/instance/internal/state"
	"github.com/giantswarm/microerror"
)

func (r *Resource) scaleUpWorkerVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Double the desired number of nodes in worker VMSS in order to
	// provide 1:1 mapping between new up-to-date nodes when draining and
	// terminating old nodes.
	nodeCountFunc := func(_ int64) int64 {
		return int64(key.WorkerCount(customObject) * 2)
	}

	err = r.scaleVMSS(ctx, customObject, key.WorkerVMSSName, nodeCountFunc)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return CordonOldWorkers, nil
}

func (r *Resource) scaleDownWorkerVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Scale down to the desired number of nodes in worker VMSS.
	nodeCountFunc := func(_ int64) int64 {
		return int64(key.WorkerCount(customObject))
	}

	err = r.scaleVMSS(ctx, customObject, key.WorkerVMSSName, nodeCountFunc)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return DeploymentCompleted, nil
}

func (r *Resource) scaleVMSS(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, nodeCountFunc func(int64) int64) error {
	c, err := r.getScaleSetsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	vmss, err := c.Get(ctx, key.ResourceGroupName(customObject), deploymentNameFunc(customObject))
	if err != nil {
		return microerror.Mask(err)
	}

	*vmss.Sku.Capacity = nodeCountFunc(*vmss.Sku.Capacity)
	res, err := c.CreateOrUpdate(ctx, key.ResourceGroupName(customObject), deploymentNameFunc(customObject), vmss)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = c.CreateOrUpdateResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
