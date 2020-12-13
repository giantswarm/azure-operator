package nodepool

import (
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/pkg/credential"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/internal/vmsku"
	"github.com/giantswarm/azure-operator/v5/service/controller/resource/nodes"
)

const (
	Name = "nodepool"
)

type Config struct {
	nodes.Config
	CredentialProvider        credential.Provider
	GSClientCredentialsConfig auth.ClientCredentialsConfig
	TenantClientFactory       tenantcluster.Factory
	VMSKU                     *vmsku.VMSKUs
}

// Resource takes care of node pool life cycle.
type Resource struct {
	nodes.Resource
	CredentialProvider  credential.Provider
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
		CredentialProvider:  config.CredentialProvider,
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
