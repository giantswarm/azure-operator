package clusterconditions

import (
	"context"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
)

const (
	MachinePoolCRsNotFoundReason = "MachinePoolCRsNotFound"
)

func (r *Resource) ensureNodePoolsReadyCondition(ctx context.Context, cluster *capi.Cluster) error {
	r.logDebug(ctx, "ensuring condition %s", aeconditions.NodePoolsReadyCondition)

	// Checking MachinePool CRs.
	machinePools, err := helpers.GetMachinePoolsByMetadata(ctx, r.ctrlClient, cluster.ObjectMeta)
	if apierrors.IsNotFound(err) {
		// We didn't find any MachinePool CRs, so NodePoolsReady is False.
		r.setMachinePoolCRsNotFound(ctx, cluster)
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	// Another check, just in case, this check is maybe redundant.
	if len(machinePools.Items) == 0 {
		// We didn't find any MachinePool CRs, so NodePoolsReady is False.
		r.setMachinePoolCRsNotFound(ctx, cluster)
		return nil
	}

	// We need a slice of Getter objects for SetAggregate, so we do a bit of
	// boxing/casting here.
	var machinePoolPointers []capiconditions.Getter
	for _, machinePool := range machinePools.Items {
		machinePoolObj := machinePool
		machinePoolPointers = append(machinePoolPointers, &machinePoolObj)
	}

	// Cluster NodePoolsReady is True when all MachinePool CRs are ready.
	capiconditions.SetAggregate(
		cluster,
		aeconditions.NodePoolsReadyCondition,
		machinePoolPointers,
		capiconditions.WithStepCounter(), // add a "x of y completed" string to the message
		capiconditions.AddSourceRef())    // add info about the originating object to the target Reason

	r.logDebug(ctx, "ensured condition %s", aeconditions.NodePoolsReadyCondition)
	return nil
}

func (r *Resource) setMachinePoolCRsNotFound(ctx context.Context, cluster *capi.Cluster) {
	const notFoundWarningMessage = "MachinePool CRs %s in namespace %s are not found"
	notFoundWarningMessageArgs := []interface{}{cluster.Name, cluster.Namespace}

	r.logWarning(ctx, notFoundWarningMessage, notFoundWarningMessageArgs...)
	capiconditions.MarkFalse(
		cluster,
		aeconditions.NodePoolsReadyCondition,
		MachinePoolCRsNotFoundReason,
		capi.ConditionSeverityWarning,
		notFoundWarningMessage,
		notFoundWarningMessageArgs...)
}
