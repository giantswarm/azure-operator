package azuremachineconditions

import (
	"context"

	creatingcondition "github.com/giantswarm/conditions-handler/pkg/conditions/creating"
	upgradingcondition "github.com/giantswarm/conditions-handler/pkg/conditions/upgrading"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiconditions "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsaware"
	"github.com/giantswarm/azure-operator/v5/pkg/azureconditions"
)

const (
	// Name is the identifier of the resource.
	Name = "azuremachineconditions"
)

type Config struct {
	WCAzureClientsFactory credentialsaware.Factory
	CtrlClient            client.Client
	Logger                micrologger.Logger
}

// Resource ensures that AzureMachinePool Status Conditions are set.
type Resource struct {
	wcAzureClientsFactory credentialsaware.Factory
	ctrlClient            client.Client
	logger                micrologger.Logger

	creatingConditionHandler  *creatingcondition.Handler
	deploymentChecker         *azureconditions.DeploymentChecker
	upgradingConditionHandler *upgradingcondition.Handler
}

func New(config Config) (*Resource, error) {
	if config.WCAzureClientsFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientsFactory must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := azureconditions.DeploymentCheckerConfig{
		CtrlClient: config.CtrlClient,
		Logger:     config.Logger,
	}
	dc, err := azureconditions.NewDeploymentChecker(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var creatingConditionHandler *creatingcondition.Handler
	{
		c := creatingcondition.HandlerConfig{
			CtrlClient:   config.CtrlClient,
			Logger:       config.Logger,
			Name:         "azureMachineCreatingHandler",
			UpdateStatus: false, // azuremachineconditions handler will do the update
		}
		creatingConditionHandler, err = creatingcondition.NewHandler(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var upgradingConditionHandler *upgradingcondition.Handler
	{
		c := upgradingcondition.HandlerConfig{
			CtrlClient:   config.CtrlClient,
			Logger:       config.Logger,
			Name:         "azureMachineUpgradingHandler",
			UpdateStatus: false, // azuremachineconditions handler will do the update
		}
		upgradingConditionHandler, err = upgradingcondition.NewHandler(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r := &Resource{
		wcAzureClientsFactory:     config.WCAzureClientsFactory,
		ctrlClient:                config.CtrlClient,
		logger:                    config.Logger,
		deploymentChecker:         dc,
		creatingConditionHandler:  creatingConditionHandler,
		upgradingConditionHandler: upgradingConditionHandler,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) logConditionStatus(ctx context.Context, azureMachine *capz.AzureMachine, conditionType capi.ConditionType) {
	condition := capiconditions.Get(azureMachine, conditionType)

	if condition == nil {
		r.logger.Debugf(ctx, "condition %s not set", conditionType)
	} else {
		messageFormat := "condition %s set to %s"
		messageArgs := []interface{}{conditionType, condition.Status}
		if condition.Status != corev1.ConditionTrue {
			messageFormat += ", Reason=%s, Severity=%s, Message=%s"
			messageArgs = append(messageArgs, condition.Reason)
			messageArgs = append(messageArgs, condition.Severity)
			messageArgs = append(messageArgs, condition.Message)
		}
		r.logger.Debugf(ctx, messageFormat, messageArgs...)
	}
}
