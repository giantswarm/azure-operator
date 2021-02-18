package client

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/client/credentialprovider"
	"github.com/giantswarm/azure-operator/v5/client/factory"
)

type CredentialsAwareClientFactoryInterface interface {
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

type CredentialsAwareClientFactory struct {
	azureCredentialProvider credentialprovider.CredentialProvider
	azureClientFactory      factory.AzureClientFactory
}

func NewCredentialsAwareClientFactory(azureCredentialProvider credentialprovider.CredentialProvider, azureClientFactory factory.AzureClientFactory) (*CredentialsAwareClientFactory, error) {
	return &CredentialsAwareClientFactory{
		azureCredentialProvider: azureCredentialProvider,
		azureClientFactory:      azureClientFactory,
	}, nil
}

func (f *CredentialsAwareClientFactory) GetLegacyCredentialSecret(ctx context.Context, organizationID string) (*v1alpha1.CredentialSecret, error) {
	legacy, err := f.azureCredentialProvider.GetLegacyCredentialSecret(ctx, organizationID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return legacy, nil
}

func (f *CredentialsAwareClientFactory) GetSubscriptionID(ctx context.Context, clusterID string) (string, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return accc.SubscriptionID, nil
}

func (f *CredentialsAwareClientFactory) GetDeploymentsClient(ctx context.Context, clusterID string) (*resources.DeploymentsClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetDeploymentsClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetDnsRecordSetsClient(ctx context.Context, clusterID string) (*dns.RecordSetsClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetDnsRecordSetsClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetGroupsClient(ctx context.Context, clusterID string) (*resources.GroupsClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetGroupsClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetInterfacesClient(ctx context.Context, clusterID string) (*network.InterfacesClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetInterfacesClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetNatGatewaysClient(ctx context.Context, clusterID string) (*network.NatGatewaysClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetNatGatewaysClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetNetworkSecurityGroupsClient(ctx context.Context, clusterID string) (*network.SecurityGroupsClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetNetworkSecurityGroupsClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetPublicIpAddressesClient(ctx context.Context, clusterID string) (*network.PublicIPAddressesClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetPublicIpAddressesClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetResourceSkusClient(ctx context.Context, clusterID string) (*compute.ResourceSkusClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetResourceSkusClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetStorageAccountsClient(ctx context.Context, clusterID string) (*storage.AccountsClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetStorageAccountsClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetSubnetsClient(ctx context.Context, clusterID string) (*network.SubnetsClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetSubnetsClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetVirtualMachineScaleSetsClient(ctx context.Context, clusterID string) (*compute.VirtualMachineScaleSetsClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetVirtualMachineScaleSetsClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, clusterID string) (*compute.VirtualMachineScaleSetVMsClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetVirtualMachineScaleSetVMsClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetVirtualNetworkGatewayConnectionsClient(ctx context.Context, clusterID string) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetVirtualNetworkGatewayConnectionsClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetVirtualNetworkGatewaysClient(ctx context.Context, clusterID string) (*network.VirtualNetworkGatewaysClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetVirtualNetworkGatewaysClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetVirtualNetworksClient(ctx context.Context, clusterID string) (*network.VirtualNetworksClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetVirtualNetworksClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func (f *CredentialsAwareClientFactory) GetZonesClient(ctx context.Context, clusterID string) (*dns.ZonesClient, error) {
	accc, err := f.azureCredentialProvider.GetAzureClientCredentialsConfig(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := f.azureClientFactory.GetZonesClient(ctx, *accc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}
