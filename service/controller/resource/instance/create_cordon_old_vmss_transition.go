package instance

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

func (r *Resource) cordonOldVMSSTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if cc.Client.TenantCluster.K8s == nil {
		r.Logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet") // nolint: errcheck
		return currentState, nil
	}

	// If the legacy VMSS still exists with at least one replica, we want to cordon its replicas.
	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Checking if the legacy VMSS %s is still present", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck
	vmss, err := r.getScaleSet(ctx, key.ResourceGroupName(cr), key.LegacyWorkerVMSSName(cr))
	if err != nil {
		return "", microerror.Mask(err)
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s is still present", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck

	// The legacy VMSS was found, check the scaling.
	legacyVmssHasInstancesRunning := *vmss.Sku.Capacity > 0
	if legacyVmssHasInstancesRunning {
		// The legacy VMSS has still instances running, cordon all of them.
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s has %d instances: cordoning those", key.LegacyWorkerVMSSName(cr), *vmss.Sku.Capacity)) // nolint: errcheck
	} else {
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("The legacy VMSS %s has 0 instances", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", "finding all worker VMSS instances") // nolint: errcheck

	var allWorkerInstances []compute.VirtualMachineScaleSetVM
	{
		allWorkerInstances, err = r.AllInstances(ctx, cr, key.LegacyWorkerVMSSName)
		if IsScaleSetNotFound(err) {
			r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("did not find the scale set '%s'", key.LegacyWorkerVMSSName(cr))) // nolint: errcheck

			return currentState, nil
		} else if err != nil {
			return "", microerror.Mask(err)
		}
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d worker VMSS instances", len(allWorkerInstances))) // nolint: errcheck
	r.Logger.LogCtx(ctx, "level", "debug", "message", "finding all tenant cluster nodes")                                     // nolint: errcheck

	var nodes []corev1.Node
	{
		nodeList, err := cc.Client.TenantCluster.K8s.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return "", microerror.Mask(err)
		}
		nodes = nodeList.Items
	}

	oldNodes, _ := sortNodesByTenantVMState(nodes, allWorkerInstances, cr, key.LegacyWorkerInstanceName)

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found %d nodes in the legacy VMSS", len(oldNodes))) // nolint: errcheck
	r.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring old nodes are cordoned")                               // nolint: errcheck

	oldNodesCordoned, err := r.ensureNodesCordoned(ctx, oldNodes)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if oldNodesCordoned < len(oldNodes) {
		r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("not all old nodes are still cordoned; %d pending", len(oldNodes)-oldNodesCordoned)) // nolint: errcheck

		return currentState, nil
	}

	r.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensured all old nodes (%d) are cordoned", oldNodesCordoned)) // nolint: errcheck

	return WaitForWorkersToBecomeReady, nil
}
