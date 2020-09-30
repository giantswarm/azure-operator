package instance

import (
	"context"
	"fmt"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/vmsscheck"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes/scalestrategy"
)

const (
	ProvisioningStateFailed    = "Failed"
	ProvisioningStateSucceeded = "Succeeded"
)

// The goal of scaleUpWorkerVMSSTransition is to double the desired number
// of nodes in worker VMSS in order to provide 1:1 mapping between new
// up-to-date nodes when draining and terminating old nodes.
// This will be done in subsequent reconciliation loops to avoid hitting the
// VMSS api too hard.
func (r *Resource) scaleUpWorkerVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	organizationAzureClientSet, err := r.OrganizationAzureClientSet.Get(ctx, &cr.ObjectMeta)
	if err != nil {
		return DeploymentUninitialized, microerror.Mask(err)
	}

	desiredWorkerCount := int64(key.WorkerCount(cr) * 2)
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The desired number of workers is: %d", desiredWorkerCount))

	currentWorkerCount, err := r.getInstancesCount(ctx, cr, key.WorkerVMSSName)
	if err != nil {
		return "", microerror.Mask(err)
	}
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The current number of workers is: %d", currentWorkerCount))

	allReady, err := vmsscheck.InstancesAreRunning(ctx, r.Logger, organizationAzureClientSet.VirtualMachineScaleSetVMsClient, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))
	if vmsscheck.IsVMSSUnsafeError(err) {
		// VMSS rate limits are not safe, let's wait for next reconciliation loop.
		return ScaleUpWorkerVMSS, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}
	// Not all workers are Running in Azure, wait for next reconciliation loop.
	if !allReady {
		return ScaleUpWorkerVMSS, nil
	}

	strategy := scalestrategy.Quick{}

	// All workers ready, we can scale up if needed.
	if desiredWorkerCount > currentWorkerCount {
		// Raise the worker count by one
		err = r.scaleVMSS(ctx, cr, key.WorkerVMSSName, desiredWorkerCount, strategy)
		if err != nil {
			return "", microerror.Mask(err)
		}

		r.InstanceWatchdog.GuardVMSS(ctx, organizationAzureClientSet.VirtualMachineScaleSetVMsClient, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))

		// Let's stay in the current state.
		return ScaleUpWorkerVMSS, nil
	}

	// We didn't scale up the VMSS, ready to move to next step.
	return CordonOldWorkers, nil
}

func (r *Resource) getInstancesCount(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string) (int64, error) {
	organizationAzureClientSet, err := r.OrganizationAzureClientSet.Get(ctx, &customObject.ObjectMeta)
	if err != nil {
		return -1, microerror.Mask(err)
	}

	vmss, err := organizationAzureClientSet.VirtualMachineScaleSetsClient.Get(ctx, key.ResourceGroupName(customObject), deploymentNameFunc(customObject))
	if err != nil {
		return 0, microerror.Mask(err)
	}

	return *vmss.Sku.Capacity, nil
}

func (r *Resource) scaleDownWorkerVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	desiredWorkerCount := int64(key.WorkerCount(cr))

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaling worker VMSS to %d nodes", desiredWorkerCount))

	strategy := scalestrategy.Quick{}

	// Scale down to the desired number of nodes in worker VMSS.
	err = r.scaleVMSS(ctx, cr, key.WorkerVMSSName, desiredWorkerCount, strategy)
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaled worker VMSS to %d nodes", desiredWorkerCount))

	return DeploymentCompleted, nil
}

func (r *Resource) scaleVMSS(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentNameFunc func(customObject providerv1alpha1.AzureConfig) string, desiredNodeCount int64, scaleStrategy scalestrategy.Interface) error {
	organizationAzureClientSet, err := r.OrganizationAzureClientSet.Get(ctx, &customObject.ObjectMeta)
	if err != nil {
		return microerror.Mask(err)
	}

	vmss, err := organizationAzureClientSet.VirtualMachineScaleSetsClient.Get(ctx, key.ResourceGroupName(customObject), deploymentNameFunc(customObject))
	if err != nil {
		return microerror.Mask(err)
	}

	computedCount := scaleStrategy.GetNodeCount(*vmss.Sku.Capacity, desiredNodeCount)

	*vmss.Sku.Capacity = computedCount
	res, err := organizationAzureClientSet.VirtualMachineScaleSetsClient.CreateOrUpdate(ctx, key.ResourceGroupName(customObject), deploymentNameFunc(customObject), vmss)
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = organizationAzureClientSet.VirtualMachineScaleSetsClient.CreateOrUpdateResponder(res.Response())
	if err != nil {
		return microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("scaled worker VMSS to %d nodes", computedCount))

	return nil
}
