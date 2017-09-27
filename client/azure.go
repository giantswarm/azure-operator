package client

import (
	"net/http"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

// AzureConfig contains the common attributes to create an Azure client.
type AzureConfig struct {
	// Dependencies.

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

// DefaultAzureConfig provides a default configuration to create an Azure client
// by best effort.
func DefaultAzureConfig() *AzureConfig {
	var err error

	var newLogger micrologger.Logger
	{
		config := micrologger.DefaultConfig()
		newLogger, err = micrologger.New(config)
		if err != nil {
			panic(err)
		}
	}

	return &AzureConfig{
		// Dependencies.
		Logger: newLogger,

		// Settings.
		ClientID:       "",
		ClientSecret:   "",
		SubscriptionID: "",
		TenantID:       "",
	}
}

// AzureClientSet is the collection of Azure API clients.
type AzureClientSet struct {
	// DeploymentsClient manages deployments of ARM templates.
	DeploymentsClient *resources.DeploymentsClient
	// GroupsClient manages ARM resource groups.
	GroupsClient *resources.GroupsClient
}

// NewAzureClientSet returns the Azure API clients.
func NewAzureClientSet(config *AzureConfig) (*AzureClientSet, error) {
	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	// Settings.
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

	groupsClient, err := newGroupsClient(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientset := &AzureClientSet{
		DeploymentsClient: deploymentsClient,
		GroupsClient:      groupsClient,
	}

	return clientset, nil
}

// ResponseWasNotFound returns true if the response code from the Azure API
// was a 404.
func ResponseWasNotFound(resp autorest.Response) bool {
	if resp.Response != nil && resp.StatusCode == http.StatusNotFound {
		return true
	}

	return false
}

func newDeploymentsClient(config *AzureConfig) (*resources.DeploymentsClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client := resources.NewDeploymentsClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newGroupsClient(config *AzureConfig) (*resources.GroupsClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client := resources.NewGroupsClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newServicePrincipalToken(config *AzureConfig, scope string) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, config.TenantID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return adal.NewServicePrincipalToken(*oauthConfig, config.ClientID, config.ClientSecret, scope)
}
