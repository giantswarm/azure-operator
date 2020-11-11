package machinepoolconditions

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Name is the identifier of the resource.
	Name = "machinepoolconditions"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// Resource ensures that MachinePool Status Conditions are set.
type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) logDebug(ctx context.Context, message string, messageArgs ...interface{}) {
	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf(message, messageArgs...))
}

func (r *Resource) logWarning(ctx context.Context, message string, messageArgs ...interface{}) {
	r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf(message, messageArgs...))
}

func (r *Resource) logConditionStatus(ctx context.Context, machinePool *capiexp.MachinePool, conditionType capi.ConditionType) {
	condition := capiconditions.Get(machinePool, conditionType)

	if condition == nil {
		r.logWarning(ctx, "condition %s not set", conditionType)
	} else {
		messageFormat := "condition %s set to %s"
		messageArgs := []interface{}{conditionType, condition.Status}
		if condition.Status != corev1.ConditionTrue {
			messageFormat += ", Reason=%s, Severity=%s, Message=%s"
			messageArgs = append(messageArgs, condition.Reason)
			messageArgs = append(messageArgs, condition.Severity)
			messageArgs = append(messageArgs, condition.Message)
		}
		r.logDebug(ctx, messageFormat, messageArgs...)
	}
}
