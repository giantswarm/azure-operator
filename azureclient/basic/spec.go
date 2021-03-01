package basic

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialprovider"
)

// The basic.Factory interface defines an interface to return an Azure API Client given the
// authentication data provided via a AzureClientCredentialsConfig object.
// Implementations of this interface are not meant to be run standalone, but to be used by a credentialawarefactory.Factory.
type Factory interface {
	GetDeploymentsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*resources.DeploymentsClient, error)
	GetDnsRecordSetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*dns.RecordSetsClient, error)
	GetGroupsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*resources.GroupsClient, error)
	GetInterfacesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.InterfacesClient, error)
	GetNatGatewaysClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.NatGatewaysClient, error)
	GetNetworkSecurityGroupsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.SecurityGroupsClient, error)
	GetPublicIpAddressesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.PublicIPAddressesClient, error)
	GetResourceSkusClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.ResourceSkusClient, error)
	GetStorageAccountsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*storage.AccountsClient, error)
	GetSubnetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.SubnetsClient, error)
	GetVirtualMachineScaleSetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetsClient, error)
	GetVirtualMachineScaleSetVMsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetVMsClient, error)
	GetVirtualNetworkGatewayConnectionsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworkGatewayConnectionsClient, error)
	GetVirtualNetworkGatewaysClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworkGatewaysClient, error)
	GetVirtualNetworksClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworksClient, error)
	GetZonesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*dns.ZonesClient, error)
}
