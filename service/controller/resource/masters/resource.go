package masters

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/resource/nodes"
)

const (
	Name = "masters"
)

type Config struct {
	nodes.Config
}

type Resource struct {
	nodes.Resource
}

func New(config Config) (*Resource, error) {
	nodes, err := nodes.New(config.Config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r := &Resource{
		*nodes,
	}
	stateMachine := r.createStateMachine()
	r.SetStateMachine(stateMachine)

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getSecurityRulesClient(ctx context.Context) (*network.SecurityRulesClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.SecurityRulesClient, nil
}
