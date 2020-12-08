package azuremachinepoolconditions

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	azureclient "github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/pkg/azureconditions"
)

const (
	// Name is the identifier of the resource.
	Name = "azuremachinepoolconditions"
)

type Config struct {
	AzureClientsFactory *azureclient.OrganizationFactory
	CtrlClient          client.Client
	Logger              micrologger.Logger
}

// Resource ensures that AzureMachinePool Status Conditions are set.
type Resource struct {
	azureClientsFactory *azureclient.OrganizationFactory
	ctrlClient          client.Client
	logger              micrologger.Logger
	deploymentChecker   *azureconditions.DeploymentChecker
}

func New(config Config) (*Resource, error) {
	if config.AzureClientsFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureClientsFactory must not be empty", config)
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

	r := &Resource{
		azureClientsFactory: config.AzureClientsFactory,
		ctrlClient:          config.CtrlClient,
		logger:              config.Logger,
		deploymentChecker:   dc,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) logDebug(ctx context.Context, message string, messageArgs ...interface{}) {
	r.logger.Debugf(ctx, message, messageArgs...)
}

func (r *Resource) logWarning(ctx context.Context, message string, messageArgs ...interface{}) {
	r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf(message, messageArgs...))
}
