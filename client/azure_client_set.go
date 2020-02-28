package client

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-07-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/giantswarm/microerror"
)

// AzureClientSet is the collection of Azure API clients.
type AzureClientSet struct {
	// The Application ID on Azure.
	ClientID string
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
	// ApplicationsClient manages Applications.
	ApplicationsClient *graphrbac.ApplicationsClient
	//StorageAccountsClient manages blobs in storage containers.
	StorageAccountsClient *storage.AccountsClient
	// The Azure Subscription ID.
	SubscriptionID string
	// UsageClient is used to work with limits and quotas.
	UsageClient *compute.UsageClient
	// VirtualNetworkClient manages virtual networks.
	VirtualNetworkClient *network.VirtualNetworksClient
	// VirtualNetworkGatewayConnectionsClient manages virtual network gateway connections.
	VirtualNetworkGatewayConnectionsClient *network.VirtualNetworkGatewayConnectionsClient
	// VirtualNetworkGatewaysClient manages virtual network gateways.
	VirtualNetworkGatewaysClient *network.VirtualNetworkGatewaysClient
	// VirtualMachineScaleSetsClient manages virtual machine scale sets.
	VirtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient
	// VirtualMachineScaleSetVMsClient manages virtual machine scale set VMs.
	VirtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient
	// VnetPeeringClient manages virtual network peerings.
	VnetPeeringClient *network.VirtualNetworkPeeringsClient
}

// NewAzureClientSet returns the Azure API clients.
func NewAzureClientSet(config AzureClientSetConfig) (*AzureClientSet, error) {
	err := config.Validate()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Returns environment object contains all API endpoints for specific Azure
	// cloud. For empty config.EnvironmentName returns Azure public cloud.
	env, err := parseAzureEnvironment(config.EnvironmentName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	serviceManagementToken, err := newServicePrincipalToken(config, env, env.ServiceManagementEndpoint)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	graphToken, err := newServicePrincipalToken(config, env, env.GraphEndpoint)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	c := &clientConfig{
		subscriptionID:          config.SubscriptionID,
		partnerIdUserAgent:      fmt.Sprintf("pid-%s", config.PartnerID),
		resourceManagerEndpoint: env.ResourceManagerEndpoint,
		serviceManagementToken:  serviceManagementToken,
		graphManagementToken:    graphToken,
	}

	deploymentsClient, err := newDeploymentsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	dnsRecordSetsClient, err := newDNSRecordSetsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	dnsZonesClient, err := newDNSZonesClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	groupsClient, err := newGroupsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	interfacesClient, err := newInterfacesClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	applicationsClient, err := newApplicationsClient(c, config.TenantID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	storageAccountsClient, err := newStorageAccountsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	usageClient, err := newUsageClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkClient, err := newVirtualNetworkClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkGatewayConnectionsClient, err := newVirtualNetworkGatewayConnectionsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkGatewaysClient, err := newVirtualNetworkGatewaysClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualMachineScaleSetVMsClient, err := newVirtualMachineScaleSetVMsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualMachineScaleSetsClient, err := newVirtualMachineScaleSetsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	vnetPeeringClient, err := newVnetPeeringClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientSet := &AzureClientSet{
		ApplicationsClient:                     applicationsClient,
		ClientID:                               config.ClientID,
		DeploymentsClient:                      deploymentsClient,
		DNSRecordSetsClient:                    dnsRecordSetsClient,
		DNSZonesClient:                         dnsZonesClient,
		GroupsClient:                           groupsClient,
		InterfacesClient:                       interfacesClient,
		StorageAccountsClient:                  storageAccountsClient,
		SubscriptionID:                         config.SubscriptionID,
		UsageClient:                            usageClient,
		VirtualNetworkClient:                   virtualNetworkClient,
		VirtualNetworkGatewayConnectionsClient: virtualNetworkGatewayConnectionsClient,
		VirtualNetworkGatewaysClient:           virtualNetworkGatewaysClient,
		VirtualMachineScaleSetVMsClient:        virtualMachineScaleSetVMsClient,
		VirtualMachineScaleSetsClient:          virtualMachineScaleSetsClient,
		VnetPeeringClient:                      vnetPeeringClient,
	}

	return clientSet, nil
}

func newDeploymentsClient(config *clientConfig) (*resources.DeploymentsClient, error) {
	c := resources.NewDeploymentsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newDNSRecordSetsClient(config *clientConfig) (*dns.RecordSetsClient, error) {
	c := dns.NewRecordSetsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newDNSZonesClient(config *clientConfig) (*dns.ZonesClient, error) {
	c := dns.NewZonesClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newGroupsClient(config *clientConfig) (*resources.GroupsClient, error) {
	c := resources.NewGroupsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newInterfacesClient(config *clientConfig) (*network.InterfacesClient, error) {
	c := network.NewInterfacesClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newApplicationsClient(config *clientConfig, tenantID string) (*graphrbac.ApplicationsClient, error) {
	c := graphrbac.NewApplicationsClient(tenantID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.graphManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newStorageAccountsClient(config *clientConfig) (*storage.AccountsClient, error) {
	c := storage.NewAccountsClient(config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newUsageClient(config *clientConfig) (*compute.UsageClient, error) {
	c := compute.NewUsageClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualNetworkClient(config *clientConfig) (*network.VirtualNetworksClient, error) {
	c := network.NewVirtualNetworksClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualNetworkGatewayConnectionsClient(config *clientConfig) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	c := network.NewVirtualNetworkGatewayConnectionsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualNetworkGatewaysClient(config *clientConfig) (*network.VirtualNetworkGatewaysClient, error) {
	c := network.NewVirtualNetworkGatewaysClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualMachineScaleSetsClient(config *clientConfig) (*compute.VirtualMachineScaleSetsClient, error) {
	c := compute.NewVirtualMachineScaleSetsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualMachineScaleSetVMsClient(config *clientConfig) (*compute.VirtualMachineScaleSetVMsClient, error) {
	c := compute.NewVirtualMachineScaleSetVMsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVnetPeeringClient(config *clientConfig) (*network.VirtualNetworkPeeringsClient, error) {
	c := network.NewVirtualNetworkPeeringsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.serviceManagementToken)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newServicePrincipalToken(config AzureClientSetConfig, env azure.Environment, resource string) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, config.TenantID)
	if err != nil {
		return nil, microerror.Maskf(err, "creating OAuth config")
	}

	token, err := adal.NewServicePrincipalToken(*oauthConfig, config.ClientID, config.ClientSecret, resource)
	if err != nil {
		return nil, microerror.Maskf(err, "getting token")
	}

	return token, nil
}

// parseAzureEnvironment returns azure environment by name.
func parseAzureEnvironment(cloudName string) (azure.Environment, error) {
	if cloudName == "" {
		return azure.PublicCloud, nil
	}

	env, err := azure.EnvironmentFromName(cloudName)
	if err != nil {
		return azure.Environment{}, microerror.Mask(err)
	}

	return env, nil
}
