// Package ingresspipname implements resource to manage migration of public IP
// resource name for public ingress loadbalancer. Originally the public IP was
// called dummy-pip, but when refactoring infrastructure so that k8s
// ingress-controller has annotation with this PIP name, it was also renamed
// more appropriately and in order to manage this with older clusters, the name
// must be kept along with migrations.
package ingresspipname

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/azure-operator/service/controller/v13/controllercontext"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
	Name = "ingresspipnamev13"
)

type Config struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger
}

type Resource struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		g8sClient: config.G8sClient,
		logger:    config.Logger,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getPIPClient(ctx context.Context) (*network.PublicIPAddressesClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.PublicIPAddressesClient, nil
}
