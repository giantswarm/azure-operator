package controllercontext

import (
	"context"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/service/controller/cloudconfig"
)

type contextKey string

const controllerKey contextKey = "controller"

type Context struct {
	MasterLBBackendPoolID string
	AzureClientSet        *client.AzureClientSet
	Client                ContextClient
	CloudConfig           cloudconfig.Interface
	ContainerURL          *azblob.ContainerURL
	MasterSubnetID        string
	Release               ContextRelease
	WorkerSubnetID        string
}

type ContextRelease struct {
	Release v1alpha1.Release
}

func (c *Context) Validate() error {
	if c.MasterLBBackendPoolID == "" {
		return microerror.Maskf(invalidContextError, "%T.MasterLBBackendPoolID must not be empty", c)
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
		return nil, microerror.Mask(notFoundError)
	}

	return c, nil
}
