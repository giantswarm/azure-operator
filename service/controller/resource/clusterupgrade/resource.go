package clusterupgrade

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
)

const (
	// Name is the identifier of the resource.
	Name = "clusterupgrade"
)

type Config struct {
	CtrlClient          client.Client
	Logger              micrologger.Logger
	TenantClientFactory tenantcluster.Factory
}

// Resource ensures that AzureCluster Status Conditions are set.
type Resource struct {
	ctrlClient          client.Client
	logger              micrologger.Logger
	tenantClientFactory tenantcluster.Factory
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.TenantClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.TenantClientFactory must not be empty", config)
	}

	r := &Resource{
		ctrlClient:          config.CtrlClient,
		logger:              config.Logger,
		tenantClientFactory: config.TenantClientFactory,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
