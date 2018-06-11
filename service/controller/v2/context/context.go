package context

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v2/cloudconfig"
)

type contextKey string

const serviceKey contextKey = "service"

type Context struct {
	AzureClientSet *client.AzureClientSet
	AzureConfig    client.AzureConfig
	CloudConfig    *cloudconfig.CloudConfig
}

func NewContext(ctx context.Context, c Context) context.Context {
	return context.WithValue(ctx, serviceKey, &c)
}

func FromContext(ctx context.Context) (*Context, error) {
	c, ok := ctx.Value(serviceKey).(*Context)
	if !ok {
		return nil, microerror.Mask(serviceNotFound)
	}

	return c, nil
}
