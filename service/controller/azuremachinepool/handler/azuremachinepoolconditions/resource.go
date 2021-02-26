package azuremachinepoolconditions

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsaware"
	"github.com/giantswarm/azure-operator/v5/pkg/azureconditions"
)

const (
	// Name is the identifier of the resource.
	Name = "azuremachinepoolconditions"
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
	deploymentChecker     *azureconditions.DeploymentChecker
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

	r := &Resource{
		wcAzureClientsFactory: config.WCAzureClientsFactory,
		ctrlClient:            config.CtrlClient,
		logger:                config.Logger,
		deploymentChecker:     dc,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
