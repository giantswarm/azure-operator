package credentialsawarefactory

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
)

// The credentialsawarefactory.Interface interface defines an interface for retrieving an Azure API Client
// Based on the Giantswarm Cluster ID. Based on the implementation, it can return Azure API Clients for the
// Management cluster of the Workload Cluster.
type Interface interface {
	GetLegacyCredentialSecret(ctx context.Context, clusterID string) (*v1alpha1.CredentialSecret, error)
	GetSubscriptionID(ctx context.Context, clusterID string) (string, error)

	GetDeploymentsClient(ctx context.Context, clusterID string) (*resources.DeploymentsClient, error)
	GetDnsRecordSetsClient(ctx context.Context, clusterID string) (*dns.RecordSetsClient, error)
	GetGroupsClient(ctx context.Context, clusterID string) (*resources.GroupsClient, error)
	GetInterfacesClient(ctx context.Context, clusterID string) (*network.InterfacesClient, error)
	GetNatGatewaysClient(ctx context.Context, clusterID string) (*network.NatGatewaysClient, error)
	GetNetworkSecurityGroupsClient(ctx context.Context, clusterID string) (*network.SecurityGroupsClient, error)
	GetPublicIpAddressesClient(ctx context.Context, clusterID string) (*network.PublicIPAddressesClient, error)
	GetResourceSkusClient(ctx context.Context, clusterID string) (*compute.ResourceSkusClient, error)
	GetStorageAccountsClient(ctx context.Context, clusterID string) (*storage.AccountsClient, error)
	GetSubnetsClient(ctx context.Context, clusterID string) (*network.SubnetsClient, error)
	GetVirtualMachineScaleSetsClient(ctx context.Context, clusterID string) (*compute.VirtualMachineScaleSetsClient, error)
	GetVirtualMachineScaleSetVMsClient(ctx context.Context, clusterID string) (*compute.VirtualMachineScaleSetVMsClient, error)
	GetVirtualNetworkGatewayConnectionsClient(ctx context.Context, clusterID string) (*network.VirtualNetworkGatewayConnectionsClient, error)
	GetVirtualNetworkGatewaysClient(ctx context.Context, clusterID string) (*network.VirtualNetworkGatewaysClient, error)
	GetVirtualNetworksClient(ctx context.Context, clusterID string) (*network.VirtualNetworksClient, error)
	GetZonesClient(ctx context.Context, clusterID string) (*dns.ZonesClient, error)
}
