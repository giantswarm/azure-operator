package client

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/client/senddecorator"
	"github.com/giantswarm/azure-operator/v4/pkg/backpressure"
)

const (
	defaultAzureGUID = "37f13270-5c7a-56ff-9211-8426baaeaabd"
)

// AzureClientSet is the collection of Azure API clients.
type AzureClientSet struct {
	// The subscription ID this client set is configured with.
	SubscriptionID string

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
	// PublicIpAddressesClient manages public IP addresses.
	PublicIpAddressesClient *network.PublicIPAddressesClient
	// SecurityRulesClient manages networking rules in a security group.
	SecurityRulesClient *network.SecurityRulesClient
	// StorageAccountsClient manages blobs in storage containers.
	StorageAccountsClient *storage.AccountsClient
	// SubnetsClient manages subnets.
	SubnetsClient *network.SubnetsClient
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

// NewAzureClientSet returns the Azure API clients using the given Authorizer.
func NewAzureClientSet(clientCredentialsConfig auth.ClientCredentialsConfig, subscriptionID, partnerID string) (*AzureClientSet, error) {
	authorizer, err := clientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if partnerID == "" {
		partnerID = defaultAzureGUID
	}
	partnerID = fmt.Sprintf("pid-%s", partnerID)

	deploymentsClient, err := newDeploymentsClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	dnsRecordSetsClient, err := newDNSRecordSetsClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	dnsZonesClient, err := newDNSZonesClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	groupsClient, err := newGroupsClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	interfacesClient, err := newInterfacesClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	publicIpAddressesClient, err := newPublicIPAddressesClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	securityRulesClient, err := newSecurityRulesClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	storageAccountsClient, err := newStorageAccountsClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	subnetsClient, err := newSubnetsClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	usageClient, err := newUsageClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkClient, err := newVirtualNetworkClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkGatewayConnectionsClient, err := newVirtualNetworkGatewayConnectionsClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkGatewaysClient, err := newVirtualNetworkGatewaysClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualMachineScaleSetVMsClient, err := newVirtualMachineScaleSetVMsClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualMachineScaleSetsClient, err := newVirtualMachineScaleSetsClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	vnetPeeringClient, err := newVnetPeeringClient(authorizer, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientSet := &AzureClientSet{
		DeploymentsClient:                      toDeploymentsClient(deploymentsClient),
		DNSRecordSetsClient:                    toDNSRecordSetsClient(dnsRecordSetsClient),
		DNSZonesClient:                         dnsZonesClient,
		GroupsClient:                           toGroupsClient(groupsClient),
		InterfacesClient:                       toInterfacesClient(interfacesClient),
		PublicIpAddressesClient:                publicIpAddressesClient,
		SecurityRulesClient:                    securityRulesClient,
		StorageAccountsClient:                  toStorageAccountsClient(storageAccountsClient),
		SubnetsClient:                          toSubnetsClient(subnetsClient),
		SubscriptionID:                         subscriptionID,
		UsageClient:                            usageClient,
		VirtualNetworkClient:                   virtualNetworkClient,
		VirtualNetworkGatewayConnectionsClient: virtualNetworkGatewayConnectionsClient,
		VirtualNetworkGatewaysClient:           virtualNetworkGatewaysClient,
		VirtualMachineScaleSetVMsClient:        toVirtualMachineScaleSetVMsClient(virtualMachineScaleSetVMsClient),
		VirtualMachineScaleSetsClient:          toVirtualMachineScaleSetsClient(virtualMachineScaleSetsClient),
		VnetPeeringClient:                      vnetPeeringClient,
	}

	return clientSet, nil
}

func prepareClient(client *autorest.Client, authorizer autorest.Authorizer, partnerID string) *autorest.Client {
	client.Authorizer = authorizer
	_ = client.AddToUserAgent(partnerID)
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, client)

	return client
}

func newDeploymentsClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (interface{}, error) {
	client := resources.NewDeploymentsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newDisksClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (interface{}, error) {
	client := compute.NewDisksClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newDNSRecordSetsClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (interface{}, error) {
	client := dns.NewRecordSetsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newDNSZonesClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (*dns.ZonesClient, error) {
	client := dns.NewZonesClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newGroupsClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (interface{}, error) {
	client := resources.NewGroupsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newInterfacesClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewInterfacesClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newPublicIPAddressesClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (*network.PublicIPAddressesClient, error) {
	client := network.NewPublicIPAddressesClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newSecurityRulesClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (*network.SecurityRulesClient, error) {
	client := network.NewSecurityRulesClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newStorageAccountsClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (interface{}, error) {
	client := storage.NewAccountsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newSubnetsClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewSubnetsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newUsageClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (*compute.UsageClient, error) {
	client := compute.NewUsageClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newVirtualNetworkClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (*network.VirtualNetworksClient, error) {
	client := network.NewVirtualNetworksClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newVirtualNetworkGatewayConnectionsClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	client := network.NewVirtualNetworkGatewayConnectionsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newVirtualNetworkGatewaysClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (*network.VirtualNetworkGatewaysClient, error) {
	client := network.NewVirtualNetworkGatewaysClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newVirtualMachineScaleSetsClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (interface{}, error) {
	client := compute.NewVirtualMachineScaleSetsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newVirtualMachineScaleSetVMsClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (interface{}, error) {
	client := compute.NewVirtualMachineScaleSetVMsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func newVnetPeeringClient(authorizer autorest.Authorizer, subscriptionID, partnerID string) (*network.VirtualNetworkPeeringsClient, error) {
	client := network.NewVirtualNetworkPeeringsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, partnerID)

	return &client, nil
}

func toDeploymentsClient(client interface{}) *resources.DeploymentsClient {
	return client.(*resources.DeploymentsClient)
}

func toDisksClient(client interface{}) *compute.DisksClient {
	return client.(*compute.DisksClient)
}

func toGroupsClient(client interface{}) *resources.GroupsClient {
	return client.(*resources.GroupsClient)
}

func toInterfacesClient(client interface{}) *network.InterfacesClient {
	return client.(*network.InterfacesClient)
}

func toVirtualMachineScaleSetsClient(client interface{}) *compute.VirtualMachineScaleSetsClient {
	return client.(*compute.VirtualMachineScaleSetsClient)
}

func toVirtualMachineScaleSetVMsClient(client interface{}) *compute.VirtualMachineScaleSetVMsClient {
	return client.(*compute.VirtualMachineScaleSetVMsClient)
}

func toDNSRecordSetsClient(client interface{}) *dns.RecordSetsClient {
	return client.(*dns.RecordSetsClient)
}

func toStorageAccountsClient(client interface{}) *storage.AccountsClient {
	return client.(*storage.AccountsClient)
}

func toSubnetsClient(client interface{}) *network.SubnetsClient {
	return client.(*network.SubnetsClient)
}
