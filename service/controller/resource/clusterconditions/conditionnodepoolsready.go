package clusterconditions

import (
	"context"
	"fmt"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
)

func (r *Resource) ensureNodePoolsReadyCondition(ctx context.Context, cluster *capi.Cluster) error {
	// Node pool CRs are not found, so we just don't use them for
	// calculating Ready condition
	const notFoundWarningMessage = "AzureMachinePool CRs %s in namespace %s are not found"
	notFoundWarningMessageArgs := []interface{}{cluster.Name, cluster.Namespace}

	// We should really check MachinePool CRs, but AzureMachinePool will do it for now.
	azureMachinePools, err := helpers.GetAzureMachinePoolsByMetadata(ctx, r.ctrlClient, cluster.ObjectMeta)
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "info", fmt.Sprintf(notFoundWarningMessage, notFoundWarningMessageArgs...))
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if len(azureMachinePools.Items) > 0 {
		r.logger.LogCtx(ctx, "level", "info", fmt.Sprintf(notFoundWarningMessage, notFoundWarningMessageArgs...))
		return nil
	}

	var azureMachinePoolPointers []capiconditions.Getter
	for _, amp := range azureMachinePools.Items {
		azureMachinePool := amp
		azureMachinePoolPointers = append(azureMachinePoolPointers, &azureMachinePool)
	}

	capiconditions.SetAggregate(
		cluster,
		aeconditions.NodePoolsReadyCondition,
		azureMachinePoolPointers,
		capiconditions.WithStepCounter(), // add a "x of y completed" string to the message
		capiconditions.AddSourceRef()) // add info about the originating object to the target Reason
	return nil
}
