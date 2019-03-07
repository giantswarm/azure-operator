package debugger

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/v5patch1/key"
)

type Config struct {
	Logger micrologger.Logger
}

type Debugger struct {
	logger micrologger.Logger
}

func New(config Config) (*Debugger, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	d := &Debugger{
		logger: config.Logger,
	}

	return d, nil
}

func (d *Debugger) LogFailedDeployment(ctx context.Context, deployment resources.DeploymentExtended) {
	if !key.IsFailedProvisioningState(*deployment.Properties.ProvisioningState) {
		return
	}

	d.logger.LogCtx(ctx,
		"correlationID", *deployment.Properties.CorrelationID,
		"id", *deployment.ID,
		"level", "error",
		"message", "deployment failed",
		"name", *deployment.Name,
	)
}
