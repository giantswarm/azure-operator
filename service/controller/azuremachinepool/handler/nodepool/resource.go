package nodepool

import (
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/pkg/handler/nodes"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/vmsku"
)

const (
	Name = "nodepool"
)

type Config struct {
	nodes.Config
	TenantClientFactory tenantcluster.Factory
	VMSKU               *vmsku.VMSKUs
}

// Resource takes care of node pool life cycle.
type Resource struct {
	nodes.Resource
	tenantClientFactory tenantcluster.Factory
	vmsku               *vmsku.VMSKUs
}

func New(config Config) (*Resource, error) {
	config.Name = Name
	nodesResource, err := nodes.New(config.Config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r := &Resource{
		Resource:            *nodesResource,
		tenantClientFactory: config.TenantClientFactory,
		vmsku:               config.VMSKU,
	}
	stateMachine := r.createStateMachine()
	r.SetStateMachine(stateMachine)

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
