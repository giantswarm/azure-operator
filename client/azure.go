package client

import (
	"errors"
	"net/http"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-09-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
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
	// SubscriptionID is the ID of the Azure subscription.
	SubscriptionID string
	// TenantID is the ID of the Active Directory tenant.
	TenantID string
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

	deploymentsClient, err := newDeploymentsClient(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	groupsClient, err := newGroupsClient(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	dnsRecordSetsClient, err := newDNSRecordSetsClient(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	dnsZonesClient, err := newDNSZonesClient(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	interfacesClient, err := newInterfacesClient(config)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vnetPeeringClient, err := newVnetPeeringClient(config)
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

func newDeploymentsClient(config AzureConfig) (*resources.DeploymentsClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Maskf(err, "creating service principal token")
	}

	client := resources.NewDeploymentsClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newGroupsClient(config AzureConfig) (*resources.GroupsClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Maskf(err, "creating service principal token")
	}

	client := resources.NewGroupsClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newDNSRecordSetsClient(config AzureConfig) (*dns.RecordSetsClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Maskf(err, "creating service principal token")
	}

	client := dns.NewRecordSetsClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newDNSZonesClient(config AzureConfig) (*dns.ZonesClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Maskf(err, "creating service principal token")
	}

	client := dns.NewZonesClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newInterfacesClient(config AzureConfig) (*network.InterfacesClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Maskf(err, "creating service principal token")
	}

	client := network.NewInterfacesClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newVnetPeeringClient(config AzureConfig) (*network.VirtualNetworkPeeringsClient, error) {
	spt, err := newServicePrincipalToken(config, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, microerror.Maskf(err, "creating service principal token")
	}

	client := network.NewVirtualNetworkPeeringsClient(config.SubscriptionID)
	client.Authorizer = autorest.NewBearerAuthorizer(spt)

	return &client, nil
}

func newServicePrincipalToken(config AzureConfig, scope string) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, config.TenantID)
	if err != nil {
		return nil, microerror.Maskf(err, "creating OAuth config")
	}

	return adal.NewServicePrincipalToken(*oauthConfig, config.ClientID, config.ClientSecret, scope)
}
