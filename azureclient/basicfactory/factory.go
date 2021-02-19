package basicfactory

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialprovider"
)

type Config struct {
	Logger micrologger.Logger
}

type BasicFactory struct {
	logger micrologger.Logger
}

func NewAzureClientFactory(c Config) (*BasicFactory, error) {
	if c.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", c)
	}
	return &BasicFactory{
		logger: c.Logger,
	}, nil
}

func (f *BasicFactory) GetDeploymentsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*resources.DeploymentsClient, error) {
	azureClient := resources.NewDeploymentsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetDnsRecordSetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*dns.RecordSetsClient, error) {
	azureClient := dns.NewRecordSetsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetGroupsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*resources.GroupsClient, error) {
	azureClient := resources.NewGroupsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetInterfacesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.InterfacesClient, error) {
	azureClient := network.NewInterfacesClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetNatGatewaysClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.NatGatewaysClient, error) {
	azureClient := network.NewNatGatewaysClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetNetworkSecurityGroupsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.SecurityGroupsClient, error) {
	azureClient := network.NewSecurityGroupsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetPublicIpAddressesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.PublicIPAddressesClient, error) {
	azureClient := network.NewPublicIPAddressesClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetResourceSkusClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.ResourceSkusClient, error) {
	azureClient := compute.NewResourceSkusClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetStorageAccountsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*storage.AccountsClient, error) {
	azureClient := storage.NewAccountsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetSubnetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.SubnetsClient, error) {
	azureClient := network.NewSubnetsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetVirtualMachineScaleSetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetsClient, error) {
	azureClient := compute.NewVirtualMachineScaleSetsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetVMsClient, error) {
	azureClient := compute.NewVirtualMachineScaleSetVMsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetVirtualNetworksClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworksClient, error) {
	azureClient := network.NewVirtualNetworksClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetVirtualNetworkGatewayConnectionsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	azureClient := network.NewVirtualNetworkGatewayConnectionsClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetVirtualNetworkGatewaysClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworkGatewaysClient, error) {
	azureClient := network.NewVirtualNetworkGatewaysClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}

func (f *BasicFactory) GetZonesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*dns.ZonesClient, error) {
	azureClient := dns.NewZonesClient(accc.SubscriptionID)
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	azureClient.Client.Authorizer = authorizer

	return &azureClient, nil
}
