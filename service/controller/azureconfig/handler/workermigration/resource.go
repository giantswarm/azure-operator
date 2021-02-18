package workermigration

import (
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/certs/v3/pkg/certs"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/controller-runtime/pkg/client"

	azureclient "github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/azureconfig/handler/workermigration/internal/azure"
)

const (
	Name = "workermigration"
)

type Config struct {
	CertsSearcher             certs.Interface
	MCAzureClientFactory      azureclient.CredentialsAwareClientFactoryInterface
	WCAzureClientFactory      azureclient.CredentialsAwareClientFactoryInterface
	CPPublicIPAddressesClient *network.PublicIPAddressesClient
	CtrlClient                client.Client
	Logger                    micrologger.Logger

	InstallationName string
	Location         string
}

type Resource struct {
	mcAzureClientFactory azureclient.CredentialsAwareClientFactoryInterface
	wcAzureClientFactory azureclient.CredentialsAwareClientFactoryInterface
	ctrlClient           client.Client
	logger               micrologger.Logger
	tenantClientFactory  tenantcluster.Factory
	wrapAzureAPI         func(cf azureclient.CredentialsAwareClientFactoryInterface, clusterID string) azure.API

	installationName string
	location         string
}

func New(config Config) (*Resource, error) {
	if config.CPPublicIPAddressesClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CPPublicIPAddressesClient must not be empty", config)
	}
	if config.MCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}
	if config.WCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstallationName must not be empty", config)
	}
	if config.Location == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Location must not be empty", config)
	}

	tenantClientFactory, err := tenantcluster.NewFactory(config.CertsSearcher, config.Logger)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cachedTenantClientFactory, err := tenantcluster.NewCachedFactory(tenantClientFactory, config.Logger)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	newResource := &Resource{
		mcAzureClientFactory: config.MCAzureClientFactory,
		wcAzureClientFactory: config.WCAzureClientFactory,
		ctrlClient:           config.CtrlClient,
		logger:               config.Logger,
		tenantClientFactory:  cachedTenantClientFactory,
		wrapAzureAPI:         azure.GetAPI,

		installationName: config.InstallationName,
		location:         config.Location,
	}

	return newResource, nil
}

func (r *Resource) Name() string {
	return Name
}
