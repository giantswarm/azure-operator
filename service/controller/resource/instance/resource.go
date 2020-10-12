package instance

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/resource/nodes"
)

const (
	Name = "instance"
)

type Config struct {
	nodes.Config
}

type Resource struct {
	nodes.Resource
}

func New(config Config) (*Resource, error) {
	config.Name = Name
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
	return r.Resource.Name()
}
