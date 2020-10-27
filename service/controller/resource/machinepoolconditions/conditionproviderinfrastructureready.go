package machinepoolconditions

import (
	"context"
	"fmt"

	aeconditions "github.com/giantswarm/apiextensions/v3/pkg/conditions"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
)

const (
	AzureMachinePoolNotFoundReason             = "AzureMachinePoolNotFound"
	AzureMachinePoolConditionReadyNotSetReason = "AzureMachinePoolConditionReadyNotSet"
)

func (r *Resource) ensureProviderInfrastructureReadyCondition(ctx context.Context, machinePool *capiexp.MachinePool) error {
	r.logDebug(ctx, "ensuring condition %s", aeconditions.ProviderInfrastructureReadyCondition)

	azureMachinePool, err := helpers.GetAzureMachinePoolByName(ctx, r.ctrlClient, machinePool.Namespace, machinePool.Name)
	if apierrors.IsNotFound(err) {
		warningMessage := "AzureMachinePool CR %s in namespace %s is not found, check again in few minutes"
		warningMessageArgs := []interface{}{machinePool.Name, machinePool.Namespace}
		r.logger.LogCtx(ctx, "level", "warning", fmt.Sprintf(warningMessage, warningMessageArgs...))

		capiconditions.MarkFalse(
			machinePool,
			aeconditions.ProviderInfrastructureReadyCondition,
			AzureMachinePoolNotFoundReason,
			capi.ConditionSeverityWarning,
			warningMessage,
			warningMessageArgs...)

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	fallbackToFalse := capiconditions.WithFallbackValue(
		false,
		AzureMachinePoolConditionReadyNotSetReason,
		capi.ConditionSeverityWarning,
		"AzureMachinePool Ready condition is not yet set, check again in few minutes")

	capiconditions.SetMirror(machinePool, aeconditions.ProviderInfrastructureReadyCondition, azureMachinePool, fallbackToFalse)
	r.logDebug(ctx, "ensured condition %s", aeconditions.ProviderInfrastructureReadyCondition)
	return nil
}
