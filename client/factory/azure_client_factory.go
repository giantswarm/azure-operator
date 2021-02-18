package factory

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

type AzureClientCredentialsConfig struct {
	ClientCredentialsConfig auth.ClientCredentialsConfig
	SubscriptionID          string
}

type AzureClientFactoryConfig struct {
	Logger micrologger.Logger
}

type AzureClientFactory struct {
	logger micrologger.Logger
}

func NewAzureClientFactory(c AzureClientFactoryConfig) (*AzureClientFactory, error) {
	if c.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", c)
	}
	return &AzureClientFactory{
		logger: c.Logger,
	}, nil
}

func (f *AzureClientFactory) GetDeploymentsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*resources.DeploymentsClient, error) {
	azureClient := resources.NewDeploymentsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetDnsRecordSetsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*dns.RecordSetsClient, error) {
	azureClient := dns.NewRecordSetsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetGroupsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*resources.GroupsClient, error) {
	azureClient := resources.NewGroupsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetInterfacesClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.InterfacesClient, error) {
	azureClient := network.NewInterfacesClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetNatGatewaysClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.NatGatewaysClient, error) {
	azureClient := network.NewNatGatewaysClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetNetworkSecurityGroupsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.SecurityGroupsClient, error) {
	azureClient := network.NewSecurityGroupsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetPublicIpAddressesClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.PublicIPAddressesClient, error) {
	azureClient := network.NewPublicIPAddressesClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetResourceSkusClient(ctx context.Context, accc AzureClientCredentialsConfig) (*compute.ResourceSkusClient, error) {
	azureClient := compute.NewResourceSkusClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetStorageAccountsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*storage.AccountsClient, error) {
	azureClient := storage.NewAccountsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetSubnetsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.SubnetsClient, error) {
	azureClient := network.NewSubnetsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetVirtualMachineScaleSetsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetsClient, error) {
	azureClient := compute.NewVirtualMachineScaleSetsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetVMsClient, error) {
	azureClient := compute.NewVirtualMachineScaleSetVMsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetVirtualNetworksClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.VirtualNetworksClient, error) {
	azureClient := network.NewVirtualNetworksClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetVirtualNetworkGatewayConnectionsClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	azureClient := network.NewVirtualNetworkGatewayConnectionsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetVirtualNetworkGatewaysClient(ctx context.Context, accc AzureClientCredentialsConfig) (*network.VirtualNetworkGatewaysClient, error) {
	azureClient := network.NewVirtualNetworkGatewaysClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *AzureClientFactory) GetZonesClient(ctx context.Context, accc AzureClientCredentialsConfig) (*dns.ZonesClient, error) {
	azureClient := dns.NewZonesClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}
