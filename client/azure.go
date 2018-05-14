package client

import (
	"errors"
	"net/http"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-09-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

type AzureConfig struct {
	Logger micrologger.Logger

	// ClientID is the ID of the Active Directory Service Principal.
	ClientID string
	// ClientSecret is the secret of the Active Directory Service Principal.
	ClientSecret string
	// The cloud environment identifier. Takes values from https://github.com/Azure/go-autorest/blob/ec5f4903f77ed9927ac95b19ab8e44ada64c1356/autorest/azure/environments.go#L13
	Cloud string
	// SubscriptionID is the ID of the Azure subscription.
	SubscriptionID string
	// TenantID is the ID of the Active Directory tenant.
	TenantID string
}

// azureClientConfig contains all essential information to create an Azure client.
type azureClientConfig struct {
	subscriptionID          string
	resourceManagerEndpoint string
	servicePrincipalToken   *adal.ServicePrincipalToken
}

func (c AzureConfig) Validate() error {
	if c.Logger == nil {
		return errors.New("Logger must not be empty")
	}

	if c.ClientID == "" {
		return errors.New("ClientID must not be empty")
	}
	if c.ClientSecret == "" {
		return errors.New("ClientSecret must not be empty")
	}
	if c.SubscriptionID == "" {
		return errors.New("SubscriptionID must not be empty")
	}
	if c.TenantID == "" {
		return errors.New("TenantID must not be empty")
	}

	return nil
}

// AzureClientSet is the collection of Azure API clients.
type AzureClientSet struct {
	// DeploymentsClient manages deployments of ARM templates.
	DeploymentsClient *resources.DeploymentsClient
	// GroupsClient manages ARM resource groups.
	GroupsClient *resources.GroupsClient
	// DNSRecordSetsClient manages DNS zones' records.
	DNSRecordSetsClient *dns.RecordSetsClient
	// DNSRecordSetsClient manages DNS zones.
	DNSZonesClient *dns.ZonesClient
	// InterfacesClient manages virtual network interfaces.
	InterfacesClient *network.InterfacesClient
	// VnetPeeringClient manages virtual network peerings.
	VnetPeeringClient *network.VirtualNetworkPeeringsClient
}

// NewAzureClientSet returns the Azure API clients.
func NewAzureClientSet(config AzureConfig) (*AzureClientSet, error) {
	if err := config.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.%s", err)
	}

	// Returns environment object contains all API endpoints for specific Azure cloud.
	// For empty config.Cloud returns Azure public cloud.
	env, err := parseAzureEnvironment(config.Cloud)
	if err != nil {
		return nil, microerror.Maskf(err, "parsing Azure environment")
	}

	servicePrincipalToken, err := newServicePrincipalToken(config, env)
	if err != nil {
		return nil, microerror.Maskf(err, "creating service principal token")
	}

	clientConfig := &azureClientConfig{
		subscriptionID:          config.SubscriptionID,
		resourceManagerEndpoint: env.ResourceManagerEndpoint,
		servicePrincipalToken:   servicePrincipalToken,
	}

	deploymentsClient, err := newDeploymentsClient(clientConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	groupsClient, err := newGroupsClient(clientConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	dnsRecordSetsClient, err := newDNSRecordSetsClient(clientConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	dnsZonesClient, err := newDNSZonesClient(clientConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	interfacesClient, err := newInterfacesClient(clientConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vnetPeeringClient, err := newVnetPeeringClient(clientConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientset := &AzureClientSet{
		DeploymentsClient:   deploymentsClient,
		GroupsClient:        groupsClient,
		DNSRecordSetsClient: dnsRecordSetsClient,
		DNSZonesClient:      dnsZonesClient,
		InterfacesClient:    interfacesClient,
		VnetPeeringClient:   vnetPeeringClient,
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

func newDeploymentsClient(config *azureClientConfig) (*resources.DeploymentsClient, error) {
	client := resources.NewDeploymentsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)

	return &client, nil
}

func newGroupsClient(config *azureClientConfig) (*resources.GroupsClient, error) {
	client := resources.NewGroupsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)

	return &client, nil
}

func newDNSRecordSetsClient(config *azureClientConfig) (*dns.RecordSetsClient, error) {
	client := dns.NewRecordSetsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)

	return &client, nil
}

func newDNSZonesClient(config *azureClientConfig) (*dns.ZonesClient, error) {
	client := dns.NewZonesClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)

	return &client, nil
}

func newInterfacesClient(config *azureClientConfig) (*network.InterfacesClient, error) {
	client := network.NewInterfacesClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)

	return &client, nil
}

func newVnetPeeringClient(config *azureClientConfig) (*network.VirtualNetworkPeeringsClient, error) {
	client := network.NewVirtualNetworkPeeringsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)

	return &client, nil
}

func newServicePrincipalToken(config AzureConfig, env azure.Environment) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, config.TenantID)
	if err != nil {
		return nil, microerror.Maskf(err, "creating OAuth config")
	}

	token, err := adal.NewServicePrincipalToken(*oauthConfig, config.ClientID, config.ClientSecret, env.ServiceManagementEndpoint)
	if err != nil {
		return nil, microerror.Maskf(err, "getting token")
	}

	return token, nil
}

// parseAzureEnvironment returns azure environment by name.
func parseAzureEnvironment(cloudName string) (azure.Environment, error) {
	var env azure.Environment
	var err error
	if cloudName == "" {
		env = azure.PublicCloud
	} else {
		env, err = azure.EnvironmentFromName(cloudName)
		if err != nil {
			return env, microerror.Maskf(err, "parsing Azure environment")
		}
	}
	return env, err
}
