package basic

import (
	"context"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialprovider"
)

// The basic.Factory interface defines an interface to return an Azure API Client given the
// authentication data provided via a AzureClientCredentialsConfig object.
// Implementations of this interface are not meant to be run standalone, but to be used by a credentialawarefactory.Factory.
type Factory interface {
	GetCredentialSecret(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*v1alpha1.CredentialSecret, error)
	GetDeploymentsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*resources.DeploymentsClient, error)
	GetGroupsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*resources.GroupsClient, error)
	GetInterfacesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.InterfacesClient, error)
	GetVirtualMachineScaleSetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetsClient, error)
	GetVirtualMachineScaleSetVMsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetVMsClient, error)
	GetVirtualNetworksClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworksClient, error)
	GetStorageAccountsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*storage.AccountsClient, error)
	GetSubnetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.SubnetsClient, error)
	GetNatGatewaysClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.NatGatewaysClient, error)
	GetResourceSkusClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.ResourceSkusClient, error)
}
