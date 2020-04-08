package client

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/giantswarm/exporterkit/collector"
	"github.com/giantswarm/microerror"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/giantswarm/azure-operator/client/senddecorator"
	"github.com/giantswarm/azure-operator/pkg/backpressure"
)

var (
	// Prometheus metrics for clients
	deploymentsClientMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_deployments",
			Help:       "HTTP metrics for Azure SDK Deployments client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	groupsClientMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_groups",
			Help:       "HTTP metrics for Azure SDK ARM Resource Groups client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	dnsRecordSetsClientMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_dnsrecordsets",
			Help:       "HTTP metrics for Azure SDK DNS Record Sets client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	dnsZonesClientMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_dnszones",
			Help:       "HTTP metrics for Azure SDK DNS Zones client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	interfacesClientMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_interfaces",
			Help:       "HTTP metrics for Azure SDK Virtual Network Interfaces client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	storageAccountsClientMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_storageaccounts",
			Help:       "HTTP metrics for Azure SDK Storage Accounts client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	usageClientMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_usage",
			Help:       "HTTP metrics for Azure SDK Usage client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	virtualNetworkMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_virtualnetwork",
			Help:       "HTTP metrics for Azure SDK Virtual Network client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	virtualNetworkGatewaysMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_virtualnetworkgateways",
			Help:       "HTTP metrics for Azure SDK Virtual Network Gateways client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	virtualNetworkGatewayConnectionsMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_virtualnetworkgatewayconnections",
			Help:       "HTTP metrics for Azure SDK Virtual Network Gateway Connections client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	virtualMachineScaleSetsMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_virtualmachinescalesets",
			Help:       "HTTP metrics for Azure SDK Virtual Machine ScaleSets client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	virtualMachineScaleSetVMsMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_virtualmachinescalesetvms",
			Help:       "HTTP metrics for Azure SDK Virtual Machine ScaleSet VMs client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)

	vnetPeeringMetricsDesc = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "azure_sdk_client_vnetpeering",
			Help:       "HTTP metrics for Azure SDK VNET Peering client.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"status_code"},
	)
)

func init() {
	// Register prometheus metrics.
	err := prometheus.Register(deploymentsClientMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(groupsClientMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(dnsRecordSetsClientMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(dnsZonesClientMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(dnsRecordSetsClientMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(interfacesClientMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(storageAccountsClientMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(usageClientMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(virtualNetworkMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(virtualNetworkGatewaysMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(virtualNetworkGatewayConnectionsMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(virtualMachineScaleSetsMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(virtualMachineScaleSetVMsMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
	err = prometheus.Register(vnetPeeringMetricsDesc)
	if err != nil && !collector.IsAlreadyRegisteredError(err) {
		panic(err)
	}
}

// AzureClientSet is the collection of Azure API clients.
type AzureClientSet struct {
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
	//StorageAccountsClient manages blobs in storage containers.
	StorageAccountsClient *storage.AccountsClient
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

// NewAzureClientSet returns the Azure API clients.
func NewAzureClientSet(config AzureClientSetConfig) (*AzureClientSet, error) {
	err := config.Validate()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Returns environment object contains all API endpoints for specific Azure
	// cloud. For empty config.EnvironmentName returns Azure public cloud.
	env, err := parseAzureEnvironment(config.EnvironmentName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	servicePrincipalToken, err := newServicePrincipalToken(config, env)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	c := &clientConfig{
		subscriptionID:          config.SubscriptionID,
		partnerIdUserAgent:      fmt.Sprintf("pid-%s", config.PartnerID),
		resourceManagerEndpoint: env.ResourceManagerEndpoint,
		servicePrincipalToken:   servicePrincipalToken,
	}

	deploymentsClient, err := newDeploymentsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	dnsRecordSetsClient, err := newDNSRecordSetsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	dnsZonesClient, err := newDNSZonesClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	groupsClient, err := newGroupsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	interfacesClient, err := newInterfacesClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	storageAccountsClient, err := newStorageAccountsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	usageClient, err := newUsageClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkClient, err := newVirtualNetworkClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkGatewayConnectionsClient, err := newVirtualNetworkGatewayConnectionsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualNetworkGatewaysClient, err := newVirtualNetworkGatewaysClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualMachineScaleSetVMsClient, err := newVirtualMachineScaleSetVMsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	virtualMachineScaleSetsClient, err := newVirtualMachineScaleSetsClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	vnetPeeringClient, err := newVnetPeeringClient(c)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	clientSet := &AzureClientSet{
		DeploymentsClient:                      deploymentsClient,
		DNSRecordSetsClient:                    dnsRecordSetsClient,
		DNSZonesClient:                         dnsZonesClient,
		GroupsClient:                           groupsClient,
		InterfacesClient:                       interfacesClient,
		StorageAccountsClient:                  storageAccountsClient,
		UsageClient:                            usageClient,
		VirtualNetworkClient:                   virtualNetworkClient,
		VirtualNetworkGatewayConnectionsClient: virtualNetworkGatewayConnectionsClient,
		VirtualNetworkGatewaysClient:           virtualNetworkGatewaysClient,
		VirtualMachineScaleSetVMsClient:        virtualMachineScaleSetVMsClient,
		VirtualMachineScaleSetsClient:          virtualMachineScaleSetsClient,
		VnetPeeringClient:                      vnetPeeringClient,
	}

	return clientSet, nil
}

func newDeploymentsClient(config *clientConfig) (*resources.DeploymentsClient, error) {
	c := resources.NewDeploymentsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, deploymentsClientMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newDNSRecordSetsClient(config *clientConfig) (*dns.RecordSetsClient, error) {
	c := dns.NewRecordSetsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, dnsRecordSetsClientMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newDNSZonesClient(config *clientConfig) (*dns.ZonesClient, error) {
	c := dns.NewZonesClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, dnsZonesClientMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newGroupsClient(config *clientConfig) (*resources.GroupsClient, error) {
	c := resources.NewGroupsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, groupsClientMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newInterfacesClient(config *clientConfig) (*network.InterfacesClient, error) {
	c := network.NewInterfacesClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, interfacesClientMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newStorageAccountsClient(config *clientConfig) (*storage.AccountsClient, error) {
	c := storage.NewAccountsClient(config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, storageAccountsClientMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newUsageClient(config *clientConfig) (*compute.UsageClient, error) {
	c := compute.NewUsageClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, usageClientMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualNetworkClient(config *clientConfig) (*network.VirtualNetworksClient, error) {
	c := network.NewVirtualNetworksClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, virtualNetworkMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualNetworkGatewayConnectionsClient(config *clientConfig) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	c := network.NewVirtualNetworkGatewayConnectionsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, virtualNetworkGatewayConnectionsMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualNetworkGatewaysClient(config *clientConfig) (*network.VirtualNetworkGatewaysClient, error) {
	c := network.NewVirtualNetworkGatewaysClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, virtualNetworkGatewaysMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualMachineScaleSetsClient(config *clientConfig) (*compute.VirtualMachineScaleSetsClient, error) {
	c := compute.NewVirtualMachineScaleSetsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, virtualMachineScaleSetsMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVirtualMachineScaleSetVMsClient(config *clientConfig) (*compute.VirtualMachineScaleSetVMsClient, error) {
	c := compute.NewVirtualMachineScaleSetVMsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, virtualMachineScaleSetVMsMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newVnetPeeringClient(config *clientConfig) (*network.VirtualNetworkPeeringsClient, error) {
	c := network.NewVirtualNetworkPeeringsClientWithBaseURI(config.resourceManagerEndpoint, config.subscriptionID)
	c.Authorizer = autorest.NewBearerAuthorizer(config.servicePrincipalToken)
	c.RetryAttempts = 1
	senddecorator.ConfigureClient(&backpressure.Backpressure{}, vnetPeeringMetricsDesc, &c.Client)
	err := c.AddToUserAgent(config.partnerIdUserAgent)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &c, nil
}

func newServicePrincipalToken(config AzureClientSetConfig, env azure.Environment) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, config.TenantID)
	if err != nil {
		return nil, microerror.Maskf(err, "creating OAuth config")
	}

	token, err := adal.NewServicePrincipalToken(*oauthConfig, config.ClientID, config.ClientSecret, env.ServiceManagementEndpoint)
	if err != nil {
		return nil, microerror.Maskf(err, "getting token")
	}

	return token, nil
}

// parseAzureEnvironment returns azure environment by name.
func parseAzureEnvironment(cloudName string) (azure.Environment, error) {
	if cloudName == "" {
		return azure.PublicCloud, nil
	}

	env, err := azure.EnvironmentFromName(cloudName)
	if err != nil {
		return azure.Environment{}, microerror.Mask(err)
	}

	return env, nil
}
