package factory

import (
	"context"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
)

type Interface interface {
	GetCredentialSecret(ctx context.Context, accc AzureClientCredentialsConfig) (*v1alpha1.CredentialSecret, error)
	GetDeploymentsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*resources.DeploymentsClient, error)
	GetGroupsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*resources.GroupsClient, error)
	GetInterfacesClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.InterfacesClient, error)
	GetVirtualMachineScaleSetsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetsClient, error)
	GetVirtualMachineScaleSetVMsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetVMsClient, error)
	GetVirtualNetworksClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.VirtualNetworksClient, error)
	GetStorageAccountsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*storage.AccountsClient, error)
	GetSubnetsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.SubnetsClient, error)
	GetNatGatewaysClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.NatGatewaysClient, error)
	GetResourceSkusClient(ctx context.Context, accc AzureClientCredentialsConfig) (*compute.ResourceSkusClient, error)
}
