package controllercontext

import (
	"context"

	"github.com/giantswarm/microerror"
)

type contextKey string

const controllerKey contextKey = "controller"

type Context struct {
	APILBBackendPoolID  string
	EtcdLBBackendPoolID string
	MasterSubnetID      string
	WorkerSubnetID      string
}

func (c *Context) Validate() error {
	if c.APILBBackendPoolID == "" {
		return microerror.Maskf(invalidContextError, "%T.APILBBackendPoolID must not be empty", c)
	}
	if c.EtcdLBBackendPoolID == "" {
		return microerror.Maskf(invalidContextError, "%T.EtcdLBBackendPoolID must not be empty", c)
	}
	if c.MasterSubnetID == "" {
		return microerror.Maskf(invalidContextError, "%T.MasterSubnetID must not be empty", c)
	}
	if c.WorkerSubnetID == "" {
		return microerror.Maskf(invalidContextError, "%T.WorkerSubnetID must not be empty", c)
	}

	return nil
}

func NewContext(ctx context.Context, c Context) context.Context {
	return context.WithValue(ctx, controllerKey, &c)
}

func FromContext(ctx context.Context) (*Context, error) {
	c, ok := ctx.Value(controllerKey).(*Context)
	if !ok {
		return nil, microerror.Maskf(notFoundError, "context key %q of type %T", controllerKey, controllerKey)
	}

	return c, nil
}
