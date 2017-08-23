package azure

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

// Config contains the common attributes to create an Azure client.
type Config struct {
	// Dependencies
	Logger micrologger.Logger

	// ClientID is the ID of the Active Directory Service Principal.
	ClientID string
	// ClientSecret is the secret of the Active Directory Service Principal.
	ClientSecret string
	// SubscriptionID is the ID of the Azure subscription.
	SubscriptionID string
	// TenantID is the ID of the Active Directory tenant.
	TenantID string
}

// DefaultConfig provides a default configuration to create an Azure client by
// best effort.
func DefaultConfig() *Config {
	var err error

	var newLogger micrologger.Logger
	{
		config := micrologger.DefaultConfig()
		newLogger, err = micrologger.New(config)
		if err != nil {
			panic(err)
		}
	}

	return &Config{
		// Dependencies.
		Logger: newLogger,

		// Settings.
		ClientID:       "",
		ClientSecret:   "",
		SubscriptionID: "",
		TenantID:       "",
	}
}

// Clients is the collection of Azure API clients.
type Clients struct {
	// DeploymentsClient manages deployments of ARM templates.
	DeploymentsClient *resources.DeploymentsClient
	// GroupClient manages ARM resource groups.
	GroupClient *resources.GroupClient
}

// NewClients returns the Azure API clients.
func NewClients(config *Config) (*Clients, error) {
	// Dependencies
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	// Settings
	if config.ClientID == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.ClientID must not be empty")
	}
	if config.ClientSecret == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.ClientSecret must not be empty")
	}
	if config.SubscriptionID == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.SubscriptionID must not be empty")
	}
	if config.TenantID == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TenantID must not be empty")
	}

	deploymentsClient, err := newDeploymentsClient(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	groupClient, err := newGroupClient(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clients := &Clients{
		DeploymentsClient: deploymentsClient,
		GroupClient:       groupClient,
	}

	return clients, nil
}

func newDeploymentsClient(config *Config) (*resources.DeploymentsClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client := resources.NewDeploymentsClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newGroupClient(config *Config) (*resources.GroupClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client := resources.NewGroupClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newServicePrincipalToken(config *Config, scope string) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, config.TenantID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return adal.NewServicePrincipalToken(*oauthConfig, config.ClientID, config.ClientSecret, scope)
}
