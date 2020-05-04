package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"

	"github.com/giantswarm/azure-operator/pkg/project"
	"github.com/giantswarm/azure-operator/service/controller/internal/state"
	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) waitNewVMSSWorkersTransition(ctx context.Context, obj interface{}, currentState state.State) (state.State, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "checking if the new VMSS workers are ready")
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Get the count of new VMSS instances.
	vmss, err := r.getScaleSet(ctx, key.ResourceGroupName(cr), key.WorkerVMSSName(cr))
	// Even in case of a NotFound error, this is unexpected and we should start from scratch.
	if err != nil {
		return "", microerror.Mask(err)
	}

	numReadyNodes, err := countReadyNodes(ctx, isNewVMSSWorker)
	if IsClientNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "tenant cluster client not available yet")
		return currentState, nil
	} else if err != nil {
		return "", microerror.Mask(err)
	}

	if numReadyNodes == 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", "There are no new VMSS workers ready. Waiting")
		return currentState, nil
	}

	if int64(numReadyNodes) != *vmss.Sku.Capacity {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Found that only %d out of the expected %d workers from the new VMSS are ready. Waiting.", numReadyNodes, *vmss.Sku.Capacity))
		return currentState, nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "New workers are all ready")

	return CordonOldVMSS, nil
}

func isNewVMSSWorker(n corev1.Node) bool {
	if !isWorker(n) {
		return false
	}

	v, ok := n.Labels[key.LabelOperatorVersion]
	if !ok {
		// The version label was not found.
		return false
	}

	return v == project.Version()
}
