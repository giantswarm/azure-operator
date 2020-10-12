package masters

import (
	"github.com/giantswarm/microerror"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/resource/nodes"
)

const (
	Name = "masters"
)

type Config struct {
	CtrlClient client.Client
	nodes.Config
}

type Resource struct {
	ctrlClient client.Client
	nodes.Resource
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}

	config.Name = Name
	nodes, err := nodes.New(config.Config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r := &Resource{
		ctrlClient: config.CtrlClient,
		Resource:   *nodes,
	}
	stateMachine := r.createStateMachine()
	r.SetStateMachine(stateMachine)

	return r, nil
}

func (r *Resource) Name() string {
	return r.Resource.Name()
}
