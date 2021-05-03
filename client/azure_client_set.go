package client

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/authorization/mgmt/authorization"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/client/senddecorator"
	"github.com/giantswarm/azure-operator/v5/pkg/backpressure"
	"github.com/giantswarm/azure-operator/v5/service/collector"
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
	DisksClient       *compute.DisksClient
	// GroupsClient manages ARM resource groups.
	GroupsClient *resources.GroupsClient
	// DNSRecordSetsClient manages DNS zones' records.
	DNSRecordSetsClient *dns.RecordSetsClient
	// DNSRecordSetsClient manages DNS zones.
	DNSZonesClient *dns.ZonesClient
	// InterfacesClient manages virtual network interfaces.
	InterfacesClient *network.InterfacesClient
	// NatGatewaysClient manages Nat Gateways.
	NatGatewaysClient *network.NatGatewaysClient
	// PublicIpAddressesClient manages public IP addresses.
	PublicIpAddressesClient *network.PublicIPAddressesClient
	// ResourceSkusClient manages VM type SKUs.
	ResourceSkusClient *compute.ResourceSkusClient
	// SecurityRulesClient manages networking rules in a security group.
	SecurityRulesClient *network.SecurityRulesClient
	SnapshotsClient     *compute.SnapshotsClient
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
func NewAzureClientSet(clientCredentialsConfig auth.ClientCredentialsConfig, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (*AzureClientSet, error) {
	authorizer, err := clientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if partnerID == "" {
		partnerID = defaultAzureGUID
	}
	partnerID = fmt.Sprintf("pid-%s", partnerID)

	deploymentsClient, err := newDeploymentsClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	disksClient, err := newDisksClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	dnsRecordSetsClient, err := newDNSRecordSetsClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	dnsZonesClient, err := newDNSZonesClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	groupsClient, err := newGroupsClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	interfacesClient, err := newInterfacesClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	natGatewaysClient, err := newNatGatewaysClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	publicIpAddressesClient, err := newPublicIPAddressesClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	resourcesSkusClient, err := newResourceSkusClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	securityRulesClient, err := newSecurityRulesClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	snapshotsClient, err := newSnapshotsClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	storageAccountsClient, err := newStorageAccountsClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	subnetsClient, err := newSubnetsClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	usageClient, err := newUsageClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkClient, err := newVirtualNetworksClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkGatewayConnectionsClient, err := newVirtualNetworkGatewayConnectionsClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkGatewaysClient, err := newVirtualNetworkGatewaysClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualMachineScaleSetVMsClient, err := newVirtualMachineScaleSetVMsClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualMachineScaleSetsClient, err := newVirtualMachineScaleSetsClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	vnetPeeringClient, err := newVnetPeeringClient(authorizer, metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientSet := &AzureClientSet{
		DeploymentsClient:                      toDeploymentsClient(deploymentsClient),
		DisksClient:                            toDisksClient(disksClient),
		DNSRecordSetsClient:                    toDNSRecordSetsClient(dnsRecordSetsClient),
		DNSZonesClient:                         dnsZonesClient,
		GroupsClient:                           toGroupsClient(groupsClient),
		InterfacesClient:                       toInterfacesClient(interfacesClient),
		NatGatewaysClient:                      toNatGatewaysClient(natGatewaysClient),
		PublicIpAddressesClient:                toPublicIPAddressesClient(publicIpAddressesClient),
		ResourceSkusClient:                     toResourceSkusClient(resourcesSkusClient),
		SecurityRulesClient:                    securityRulesClient,
		SnapshotsClient:                        toSnapshotsClient(snapshotsClient),
		StorageAccountsClient:                  toStorageAccountsClient(storageAccountsClient),
		SubnetsClient:                          toSubnetsClient(subnetsClient),
		SubscriptionID:                         subscriptionID,
		UsageClient:                            usageClient,
		VirtualNetworkClient:                   toVirtualNetworksClient(virtualNetworkClient),
		VirtualNetworkGatewayConnectionsClient: toVirtualNetworkGatewayConnectionsClient(virtualNetworkGatewayConnectionsClient),
		VirtualNetworkGatewaysClient:           toVirtualNetworkGatewaysClient(virtualNetworkGatewaysClient),
		VirtualMachineScaleSetVMsClient:        toVirtualMachineScaleSetVMsClient(virtualMachineScaleSetVMsClient),
		VirtualMachineScaleSetsClient:          toVirtualMachineScaleSetsClient(virtualMachineScaleSetsClient),
		VnetPeeringClient:                      toVirtualNetworkPeeringsClient(vnetPeeringClient),
	}

	return clientSet, nil
}

func prepareClient(client *autorest.Client, authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, name, subscriptionID, partnerID string) *autorest.Client {
	client.Authorizer = authorizer
	_ = client.AddToUserAgent(partnerID)
	senddecorator.WrapClient(client,
		// Rate limit circuit breaker should be first so that it shortcuts the
		// request before metrics measurements. Otherwise the request metrics
		// would be skewed by sub-millisecond roundtrips.
		senddecorator.RateLimitCircuitBreaker(&backpressure.Backpressure{}),

		// Gather metrics from API calls.
		senddecorator.MetricsDecorator(name, subscriptionID, metricsCollector),
	)

	return client
}

func newDeploymentsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := resources.NewDeploymentsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "deployments", subscriptionID, partnerID)

	return &client, nil
}

func newDisksClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := compute.NewDisksClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "disks", subscriptionID, partnerID)

	return &client, nil
}

func newDNSRecordSetsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := dns.NewRecordSetsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "dns_record_sets", subscriptionID, partnerID)

	return &client, nil
}

func newDNSZonesClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (*dns.ZonesClient, error) {
	client := dns.NewZonesClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "dns_zones", subscriptionID, partnerID)

	return &client, nil
}

func newGroupsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := resources.NewGroupsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "groups", subscriptionID, partnerID)

	return &client, nil
}

func newInterfacesClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewInterfacesClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "interfaces", subscriptionID, partnerID)

	return &client, nil
}

func newNatGatewaysClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewNatGatewaysClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "nat_gateways", subscriptionID, partnerID)

	return &client, nil
}

func newNetworkSecurityGroupsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewSecurityGroupsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "network_security_groups", subscriptionID, partnerID)

	return &client, nil
}

func newPublicIPAddressesClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewPublicIPAddressesClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "public_ip_addresses", subscriptionID, partnerID)

	return &client, nil
}

func newSecurityRulesClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (*network.SecurityRulesClient, error) {
	client := network.NewSecurityRulesClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "security_rules", subscriptionID, partnerID)

	return &client, nil
}

func newSnapshotsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := compute.NewSnapshotsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "snapshots", subscriptionID, partnerID)

	return &client, nil
}

func newStorageAccountsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := storage.NewAccountsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "storage_accounts", subscriptionID, partnerID)

	return &client, nil
}

func newSubnetsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewSubnetsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "subnets", subscriptionID, partnerID)

	return &client, nil
}

func newUsageClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (*compute.UsageClient, error) {
	client := compute.NewUsageClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "usage", subscriptionID, partnerID)

	return &client, nil
}

func newVirtualNetworksClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewVirtualNetworksClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "virtual_networks", subscriptionID, partnerID)

	return &client, nil
}

func newVirtualNetworkGatewayConnectionsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewVirtualNetworkGatewayConnectionsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "virtual_network_gateway_connections", subscriptionID, partnerID)

	return &client, nil
}

func newVirtualNetworkGatewaysClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewVirtualNetworkGatewaysClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "virtual_network_gateways", subscriptionID, partnerID)

	return &client, nil
}

func newVirtualMachineScaleSetsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := compute.NewVirtualMachineScaleSetsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "virtual_machine_scale_sets", subscriptionID, partnerID)

	return &client, nil
}

func newVirtualMachineScaleSetVMsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := compute.NewVirtualMachineScaleSetVMsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "virtual_machine_scale_set_vms", subscriptionID, partnerID)

	return &client, nil
}

func newVnetPeeringClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := network.NewVirtualNetworkPeeringsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "vnet_peering", subscriptionID, partnerID)

	return &client, nil
}

func newResourceSkusClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := compute.NewResourceSkusClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "resource_skus", subscriptionID, partnerID)

	return &client, nil
}

func newRoleAssignmentsClient(authorizer autorest.Authorizer, metricsCollector collector.AzureAPIMetrics, subscriptionID, partnerID string) (interface{}, error) {
	client := authorization.NewRoleAssignmentsClient(subscriptionID)
	prepareClient(&client.Client, authorizer, metricsCollector, "role_assignments", subscriptionID, partnerID)

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

func toVirtualNetworksClient(client interface{}) *network.VirtualNetworksClient {
	return client.(*network.VirtualNetworksClient)
}

func toDNSRecordSetsClient(client interface{}) *dns.RecordSetsClient {
	return client.(*dns.RecordSetsClient)
}

func toSnapshotsClient(client interface{}) *compute.SnapshotsClient {
	return client.(*compute.SnapshotsClient)
}

func toStorageAccountsClient(client interface{}) *storage.AccountsClient {
	return client.(*storage.AccountsClient)
}

func toSubnetsClient(client interface{}) *network.SubnetsClient {
	return client.(*network.SubnetsClient)
}

func toNatGatewaysClient(client interface{}) *network.NatGatewaysClient {
	return client.(*network.NatGatewaysClient)
}

func toNetworkSecurityGroupsClient(client interface{}) *network.SecurityGroupsClient {
	return client.(*network.SecurityGroupsClient)
}

func toPublicIPAddressesClient(client interface{}) *network.PublicIPAddressesClient {
	return client.(*network.PublicIPAddressesClient)
}

func toResourceSkusClient(client interface{}) *compute.ResourceSkusClient {
	return client.(*compute.ResourceSkusClient)
}

func toVirtualNetworkPeeringsClient(client interface{}) *network.VirtualNetworkPeeringsClient {
	return client.(*network.VirtualNetworkPeeringsClient)
}

func toVirtualNetworkGatewaysClient(client interface{}) *network.VirtualNetworkGatewaysClient {
	return client.(*network.VirtualNetworkGatewaysClient)
}

func toVirtualNetworkGatewayConnectionsClient(client interface{}) *network.VirtualNetworkGatewayConnectionsClient {
	return client.(*network.VirtualNetworkGatewayConnectionsClient)
}

func toRoleAssignmentsClient(client interface{}) *authorization.RoleAssignmentsClient {
	return client.(*authorization.RoleAssignmentsClient)
}
