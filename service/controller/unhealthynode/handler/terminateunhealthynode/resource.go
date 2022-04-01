package terminateunhealthynode

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/tenantcluster/v6/pkg/tenantcluster"
	"sigs.k8s.io/controller-runtime/pkg/client"

	azureclient "github.com/giantswarm/azure-operator/v5/client"
)

const (
	Name = "terminateunhealthynode"
)

type Config struct {
	AzureClientsFactory      *azureclient.OrganizationFactory
	CtrlClient               client.Client
	Logger                   micrologger.Logger
	TenantRestConfigProvider *tenantcluster.TenantCluster
}

type Resource struct {
	azureClientsFactory      *azureclient.OrganizationFactory
	ctrlClient               client.Client
	logger                   micrologger.Logger
	tenantRestConfigProvider *tenantcluster.TenantCluster
}

func New(config Config) (*Resource, error) {
	if config.AzureClientsFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureClientsFactory must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.TenantRestConfigProvider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.TenantRestConfigProvider must not be empty", config)
	}

	r := &Resource{
		azureClientsFactory:      config.AzureClientsFactory,
		ctrlClient:               config.CtrlClient,
		logger:                   config.Logger,
		tenantRestConfigProvider: config.TenantRestConfigProvider,
	}

	return r, nil
}

func (r *Resource) Name() string {
	return Name
}
