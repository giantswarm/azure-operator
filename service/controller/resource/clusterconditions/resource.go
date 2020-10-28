package clusterconditions

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Name is the identifier of the resource.
	Name = "clusterconditions"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// Resource ensures that Cluster Status Conditions are set.
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