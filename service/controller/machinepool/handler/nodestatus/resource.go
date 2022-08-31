package nodestatus

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v6/pkg/tenantcluster"
)

const (
	// Name is the identifier of the resource.
	Name = "nodestatus"
)

type Config struct {
	CtrlClient          ctrlclient.Client
	Logger              micrologger.Logger
	TenantClientFactory tenantcluster.Factory
}

// Resource updates the MachinePool status field with the Nodes status.
type Resource struct {
	ctrlClient          ctrlclient.Client
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

func (r *Resource) Name() string {
	return Name
}
