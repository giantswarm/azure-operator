package client

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/authorization/mgmt/authorization"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	gocache "github.com/patrickmn/go-cache"

	"github.com/giantswarm/azure-operator/v5/pkg/credential"
	"github.com/giantswarm/azure-operator/v5/service/collector"
)

const (
	cacheHitLogKey       = "cacheHit"
	clientTypeLogKey     = "clientType"
	credentialNameLogKey = "credentialName"
	logLevelLogKey       = "level"
	logLevelDebug        = "debug"
	messageLogKey        = "message"
)

type FactoryConfig struct {
	AzureAPIMetrics    collector.AzureAPIMetrics
	CacheDuration      time.Duration
	CredentialProvider credential.Provider
	Logger             micrologger.Logger
}

// Factory is creating Azure clients for specified AzureConfig CRs, so basically for specified
// tenant clusters. All created clients are cached.
type Factory struct {
	credentialProvider credential.Provider
	logger             micrologger.Logger
	metricsCollector   collector.AzureAPIMetrics
	mutex              sync.Mutex

	// map [credentialName + client type] -> client
	cachedClients *gocache.Cache
}

type clientCreatorFunc func(autorest.Authorizer, collector.AzureAPIMetrics, string, string) (interface{}, error)

// NewFactory returns a new Azure client factory that is used throughout entire azure-operator
// lifetime.
func NewFactory(config FactoryConfig) (*Factory, error) {
	if config.AzureAPIMetrics == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureAPIMetrics must not be empty", config)
	}
	if config.CacheDuration < 5*time.Minute { // cache at least for one reconciliation loop duration
		return nil, microerror.Maskf(invalidConfigError, "%T.CacheDuration must be at least 5 minutes", config)
	}
	if config.CredentialProvider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CredentialProvider must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	factory := &Factory{
		logger:             config.Logger,
		credentialProvider: config.CredentialProvider,
		cachedClients:      gocache.New(config.CacheDuration, 2*config.CacheDuration),
		metricsCollector:   config.AzureAPIMetrics,
	}

	factory.cachedClients.OnEvicted(func(clientKey string, i interface{}) {
		factory.onEvicted(clientKey)
	})

	return factory, nil
}

// GetDeploymentsClient returns DeploymentsClient that is used for management of deployments and
// ARM templates. The client (for specified cluster) is cached after creation, so the same client
// is returned every time.
func (f *Factory) GetDeploymentsClient(credentialNamespace, credentialName string) (*resources.DeploymentsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "DeploymentsClient", newDeploymentsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toDeploymentsClient(client), nil
}

// GetDisksClient returns DisksClient that is used for management of virtual disks.
// The client (for specified cluster) is cached after creation, so the same client
// is returned every time.
func (f *Factory) GetDisksClient(credentialNamespace, credentialName string) (*compute.DisksClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "DisksClient", newDisksClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toDisksClient(client), nil
}

// GetGroupsClient returns GroupsClient that is used for management of resource groups for the
// specified cluster. The created client is cached for the time period specified in the factory
// config.
func (f *Factory) GetGroupsClient(credentialNamespace, credentialName string) (*resources.GroupsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "GroupsClient", newGroupsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toGroupsClient(client), nil
}

// GetInterfacesClient returns InterfacesClient that is used for management of network interfaces for the
// specified cluster. The created client is cached for the time period specified in the factory
// config.
func (f *Factory) GetInterfacesClient(credentialNamespace, credentialName string) (*network.InterfacesClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "InterfacesClient", newInterfacesClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toInterfacesClient(client), nil
}

// GetDNSRecordSetsClient returns RecordSetsClient that is used for management of DNS records.
// The client (for specified cluster) is cached after creation, so the same client
// is returned every time.
func (f *Factory) GetDNSRecordSetsClient(credentialNamespace, credentialName string) (*dns.RecordSetsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "RecordSetsClient", newDNSRecordSetsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toDNSRecordSetsClient(client), nil
}

// GetVirtualMachineScaleSetsClient returns VirtualMachineScaleSetsClient that is used for
// management of virtual machine scale sets for the specified cluster. The created client is cached
// for the time period specified in the factory config.
func (f *Factory) GetVirtualMachineScaleSetsClient(credentialNamespace, credentialName string) (*compute.VirtualMachineScaleSetsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "VirtualMachineScaleSetsClient", newVirtualMachineScaleSetsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toVirtualMachineScaleSetsClient(client), nil
}

// GetVirtualMachineScaleSetVMsClient returns GetVirtualMachineScaleSetVMsClient that is used for
// management of virtual machine scale set instances for the specified cluster. The created client
// is cached for the time period specified in the factory config.
func (f *Factory) GetVirtualMachineScaleSetVMsClient(credentialNamespace, credentialName string) (*compute.VirtualMachineScaleSetVMsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "VirtualMachineScaleSetVMsClient", newVirtualMachineScaleSetVMsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toVirtualMachineScaleSetVMsClient(client), nil
}

// GetVirtualNetworksClient returns *network.VirtualNetworksClient that is used for management of Azure
// virtual networks. The created client is cached for the time period specified in the factory config.
func (f *Factory) GetVirtualNetworksClient(credentialNamespace, credentialName string) (*network.VirtualNetworksClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "VirtualNetworksClient", newVirtualNetworksClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toVirtualNetworksClient(client), nil
}

// GetSnapshotsClient returns *compute.SnapshotsClient that is used for management of Azure
// snapshots (both disks and VMs). The created client is cached for the time period
// specified in the factory config.
func (f *Factory) GetSnapshotsClient(credentialNamespace, credentialName string) (*compute.SnapshotsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "SnapshotsClient", newSnapshotsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toSnapshotsClient(client), nil
}

// GetStorageAccountsClient returns *storage.AccountsClient that is used for management of Azure
// storage accounts for the specified cluster. The created client is cached for the time period
// specified in the factory config.
func (f *Factory) GetStorageAccountsClient(credentialNamespace, credentialName string) (*storage.AccountsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "StorageAccountsClient", newStorageAccountsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toStorageAccountsClient(client), nil
}

// GetSubnetsClient returns *network.SubnetsClient that is used for management of Azure subnets.
// The created client is cached for the time period specified in the factory config.
func (f *Factory) GetSubnetsClient(credentialNamespace, credentialName string) (*network.SubnetsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "SubnetsClient", newSubnetsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toSubnetsClient(client), nil
}

// GetNatGatewaysClient returns *network.NatGatewaysClient that is used for management of Nat Gateways.
// The created client is cached for the time period specified in the factory config.
func (f *Factory) GetNatGatewaysClient(credentialNamespace, credentialName string) (*network.NatGatewaysClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "NatGatewaysClient", newNatGatewaysClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toNatGatewaysClient(client), nil
}

// GetNetworkSecurityGroupsClient returns *network.SecurityGroupsClient that is used for management of Network Security Groups.
// The created client is cached for the time period specified in the factory config.
func (f *Factory) GetNetworkSecurityGroupsClient(credentialNamespace, credentialName string) (*network.SecurityGroupsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "NetworkSecurityGroupsClient", newNetworkSecurityGroupsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toNetworkSecurityGroupsClient(client), nil
}

func (f *Factory) GetPublicIPAddressesClient(credentialNamespace, credentialName string) (*network.PublicIPAddressesClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "PublicIPAddressesClient", newPublicIPAddressesClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toPublicIPAddressesClient(client), nil
}

// GetResourceSkusClient returns *compute.ResourceSkusClient that is used for reading VM instance types.
// The created client is cached for the time period specified in the factory config.
func (f *Factory) GetResourceSkusClient(credentialNamespace, credentialName string) (*compute.ResourceSkusClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "ResourceSkusClient", newResourceSkusClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toResourceSkusClient(client), nil
}

func (f *Factory) GetVirtualNetworkPeeringsClient(credentialNamespace, credentialName string) (*network.VirtualNetworkPeeringsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "VirtualNetworkPeeringsClient", newVnetPeeringClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toVirtualNetworkPeeringsClient(client), nil
}

func (f *Factory) GetVirtualNetworkGatewaysClient(credentialNamespace, credentialName string) (*network.VirtualNetworkGatewaysClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "VirtualNetworkGatewaysClient", newVirtualNetworkGatewaysClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toVirtualNetworkGatewaysClient(client), nil
}

func (f *Factory) GetVirtualNetworkGatewayConnectionsClient(credentialNamespace, credentialName string) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "VirtualNetworkGatewayConnectionsClient", newVirtualNetworkGatewayConnectionsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toVirtualNetworkGatewayConnectionsClient(client), nil
}

func (f *Factory) GetRoleAssignmentsClient(credentialNamespace, credentialName string) (*authorization.RoleAssignmentsClient, error) {
	client, err := f.getClient(credentialNamespace, credentialName, "RoleAssignmentsClient", newRoleAssignmentsClient)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return toRoleAssignmentsClient(client), nil
}

func (f *Factory) getClient(credentialNamespace, credentialName string, clientType string, createClient clientCreatorFunc) (interface{}, error) {
	l := f.logger.With(
		logLevelLogKey, logLevelDebug,
		messageLogKey, "get client",
		credentialNameLogKey, credentialName,
		clientTypeLogKey, clientType)

	clientKey := getClientKey(credentialName, clientType)
	var client interface{}
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if cachedClient, ok := f.cachedClients.Get(clientKey); ok {
		// client found, it will be refreshed in cache
		l.Log(cacheHitLogKey, true)
		client = cachedClient
	} else {
		// client not found, create it, it will be saved in cache
		l.Log(cacheHitLogKey, false)
		newClient, err := f.createClient(credentialNamespace, credentialName, createClient)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		client = newClient
	}

	// refresh existing client or set new one
	f.cachedClients.SetDefault(clientKey, client)
	return client, nil
}

func (f *Factory) createClient(credentialNamespace, credentialName string, createClient clientCreatorFunc) (interface{}, error) {
	organizationCredentialsConfig, subscriptionID, partnerID, err := f.credentialProvider.GetOrganizationAzureCredentials(context.Background(), credentialNamespace, credentialName)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	authorizer, err := organizationCredentialsConfig.Authorizer()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	client, err := createClient(authorizer, f.metricsCollector, subscriptionID, partnerID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client, nil
}

func getClientKey(credentialName string, clientType string) string {
	return fmt.Sprintf("%s.%s", credentialName, clientType)
}

func getClientKeyParts(clientKey string) (credentialName, clientType string) {
	parts := strings.Split(clientKey, ".")
	partsCount := len(parts)

	if partsCount > 2 {
		credentialName = strings.Join(parts[0:partsCount-1], ".")
		clientType = parts[partsCount-1]
	} else if partsCount == 2 {
		credentialName, clientType = parts[0], parts[1]
	} else {
		// this should never happen, don't return error, this is for logging only
		credentialName, clientType = "unknown", "unknown"
	}

	return
}

func (f *Factory) onEvicted(clientKey string) {
	credentialName, clientType := getClientKeyParts(clientKey)
	f.logger.Log(
		logLevelLogKey, logLevelDebug,
		messageLogKey, "client evicted",
		credentialNameLogKey, credentialName,
		clientTypeLogKey, clientType)
}
