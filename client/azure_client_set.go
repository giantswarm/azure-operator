package client

import (
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-09-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/giantswarm/microerror"
)

type AzureClientSetConfig struct {
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

// ClientConfig contains all essential information to create an Azure client.
type ClientConfig struct {
	SubscriptionID          string
	ResourceManagerEndpoint string
	ServicePrincipalToken   *adal.ServicePrincipalToken
}

func (c AzureClientSetConfig) Validate() error {
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
	// VirtualNetworkClient manages virtual networks.
	VirtualNetworkClient *network.VirtualNetworksClient
	// VirtualNetworkGatewaysClient manages virtual network gateways.
	VirtualNetworkGatewaysClient *network.VirtualNetworkGatewaysClient
	// VirtualNetworkGatewayConnectionsClient manages virtual network gateway connections.
	VirtualNetworkGatewayConnectionsClient *network.VirtualNetworkGatewayConnectionsClient
	// VirtualMachineScaleSetsClient manages virtual machine scale sets.
	VirtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient
	// VirtualMachineScaleSetVMsClient manages virtual machine scale set VMs.
	VirtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient
	// VnetPeeringClient manages virtual network peerings.
	VnetPeeringClient *network.VirtualNetworkPeeringsClient
}

// NewAzureClientSet returns the Azure API clients.
func NewAzureClientSet(config AzureClientSetConfig) (*AzureClientSet, error) {
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

	c := &ClientConfig{
		SubscriptionID:          config.SubscriptionID,
		ResourceManagerEndpoint: env.ResourceManagerEndpoint,
		ServicePrincipalToken:   servicePrincipalToken,
	}

	clientSet := &AzureClientSet{
		DeploymentsClient:                      newDeploymentsClient(c),
		GroupsClient:                           newGroupsClient(c),
		DNSRecordSetsClient:                    newDNSRecordSetsClient(c),
		DNSZonesClient:                         newDNSZonesClient(c),
		InterfacesClient:                       newInterfacesClient(c),
		VirtualNetworkClient:                   newVirtualNetworkClient(c),
		VirtualNetworkGatewaysClient:           newVirtualNetworkGatewaysClient(c),
		VirtualNetworkGatewayConnectionsClient: newVirtualNetworkGatewayConnectionsClient(c),
		VirtualMachineScaleSetVMsClient:        newVirtualMachineScaleSetVMsClient(c),
		VirtualMachineScaleSetsClient:          newVirtualMachineScaleSetsClient(c),
		VnetPeeringClient:                      newVnetPeeringClient(c),
	}

	return clientSet, nil
}

// ResponseWasNotFound returns true if the response code from the Azure API
// was a 404.
func ResponseWasNotFound(resp autorest.Response) bool {
	if resp.Response != nil && resp.StatusCode == http.StatusNotFound {
		return true
	}

	return false
}

func newDeploymentsClient(config *ClientConfig) *resources.DeploymentsClient {
	c := resources.NewDeploymentsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newGroupsClient(config *ClientConfig) *resources.GroupsClient {
	c := resources.NewGroupsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newDNSRecordSetsClient(config *ClientConfig) *dns.RecordSetsClient {
	c := dns.NewRecordSetsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newDNSZonesClient(config *ClientConfig) *dns.ZonesClient {
	c := dns.NewZonesClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newInterfacesClient(config *ClientConfig) *network.InterfacesClient {
	c := network.NewInterfacesClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newVirtualNetworkClient(config *ClientConfig) *network.VirtualNetworksClient {
	c := network.NewVirtualNetworksClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newVirtualNetworkGatewaysClient(config *ClientConfig) *network.VirtualNetworkGatewaysClient {
	c := network.NewVirtualNetworkGatewaysClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newVirtualNetworkGatewayConnectionsClient(config *ClientConfig) *network.VirtualNetworkGatewayConnectionsClient {
	c := network.NewVirtualNetworkGatewayConnectionsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newVirtualMachineScaleSetsClient(config *ClientConfig) *compute.VirtualMachineScaleSetsClient {
	c := compute.NewVirtualMachineScaleSetsClient(config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newVirtualMachineScaleSetVMsClient(config *ClientConfig) *compute.VirtualMachineScaleSetVMsClient {
	c := compute.NewVirtualMachineScaleSetVMsClient(config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newVnetPeeringClient(config *ClientConfig) *network.VirtualNetworkPeeringsClient {
	c := network.NewVirtualNetworkPeeringsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.ServicePrincipalToken)

	return &c
}

func newServicePrincipalToken(config AzureClientSetConfig, env azure.Environment) (*adal.ServicePrincipalToken, error) {
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
	return env, microerror.Maskf(err, "parsing Azure environment")
}
