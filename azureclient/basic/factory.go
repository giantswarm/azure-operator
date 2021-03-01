package basic

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialprovider"
	"github.com/giantswarm/azure-operator/v5/azureclient/senddecorator"
	"github.com/giantswarm/azure-operator/v5/pkg/backpressure"
	"github.com/giantswarm/azure-operator/v5/service/collector"
)

type Config struct {
	Logger           micrologger.Logger
	MetricsCollector collector.AzureAPIMetrics
	PartnerID        string
}

type factory struct {
	logger           micrologger.Logger
	metricsCollector collector.AzureAPIMetrics
	partnerID        string
}

func New(c Config) (Factory, error) {
	if c.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", c)
	}
	return &factory{
		logger:           c.Logger,
		partnerID:        c.PartnerID,
		metricsCollector: c.MetricsCollector,
	}, nil
}

func (f *factory) GetDeploymentsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*resources.DeploymentsClient, error) {
	azureClient := resources.NewDeploymentsClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "deployments")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetDnsRecordSetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*dns.RecordSetsClient, error) {
	azureClient := dns.NewRecordSetsClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "dns_record_sets")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetGroupsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*resources.GroupsClient, error) {
	azureClient := resources.NewGroupsClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "groups")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetInterfacesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.InterfacesClient, error) {
	azureClient := network.NewInterfacesClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "interfaces")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetNatGatewaysClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.NatGatewaysClient, error) {
	azureClient := network.NewNatGatewaysClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "nat_gateways")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetNetworkSecurityGroupsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.SecurityGroupsClient, error) {
	azureClient := network.NewSecurityGroupsClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "network_security_groups")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetPublicIpAddressesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.PublicIPAddressesClient, error) {
	azureClient := network.NewPublicIPAddressesClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "public_ip_addresses")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetResourceSkusClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.ResourceSkusClient, error) {
	azureClient := compute.NewResourceSkusClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "resource_skus")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetStorageAccountsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*storage.AccountsClient, error) {
	azureClient := storage.NewAccountsClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "storage_accounts")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetSubnetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.SubnetsClient, error) {
	azureClient := network.NewSubnetsClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "subnets")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetVirtualMachineScaleSetsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetsClient, error) {
	azureClient := compute.NewVirtualMachineScaleSetsClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "virtual_machine_scale_sets")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*compute.VirtualMachineScaleSetVMsClient, error) {
	azureClient := compute.NewVirtualMachineScaleSetVMsClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "virtual_machine_scale_set_vms")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetVirtualNetworksClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworksClient, error) {
	azureClient := network.NewVirtualNetworksClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "virtual_networks")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetVirtualNetworkGatewayConnectionsClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	azureClient := network.NewVirtualNetworkGatewayConnectionsClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "virtual_network_gateway_connections")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetVirtualNetworkGatewaysClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*network.VirtualNetworkGatewaysClient, error) {
	azureClient := network.NewVirtualNetworkGatewaysClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "virtual_network_gateways")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) GetZonesClient(ctx context.Context, accc credentialprovider.AzureClientCredentialsConfig) (*dns.ZonesClient, error) {
	azureClient := dns.NewZonesClient(accc.SubscriptionID)

	err := f.prepareClient(&azureClient.Client, accc, "dns_zones")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &azureClient, nil
}

func (f *factory) prepareClient(client *autorest.Client, accc credentialprovider.AzureClientCredentialsConfig, name string) error {
	authorizer, err := accc.ClientCredentialsConfig.Authorizer()
	if err != nil {
		return microerror.Mask(err)
	}

	client.Authorizer = authorizer
	if f.partnerID != "" {
		_ = client.AddToUserAgent(f.partnerID)
	}
	decorators := []autorest.SendDecorator{
		// Rate limit circuit breaker should be first so that it shortcuts the
		// request before metrics measurements. Otherwise the request metrics
		// would be skewed by sub-millisecond roundtrips.
		senddecorator.RateLimitCircuitBreaker(&backpressure.Backpressure{}),
	}

	// Gather metrics from API calls.
	if f.metricsCollector != nil {
		decorators = append(decorators, senddecorator.MetricsDecorator(name, accc.SubscriptionID, f.metricsCollector))
	}

	senddecorator.WrapClient(client, decorators...)

	return nil
}
