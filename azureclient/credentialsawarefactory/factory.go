package credentialsawarefactory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	gocache "github.com/patrickmn/go-cache"

	"github.com/giantswarm/azure-operator/v5/azureclient/basicfactory"
	"github.com/giantswarm/azure-operator/v5/azureclient/credentialprovider"
)

type CredentialsAwareClientFactory struct {
	azureCredentialProvider credentialprovider.CredentialProvider
	azureClientFactory      basicfactory.BasicFactory

	cachedClients *gocache.Cache
	mutex         sync.Mutex
}

func NewCredentialsAwareClientFactory(azureCredentialProvider credentialprovider.CredentialProvider, azureClientFactory basicfactory.BasicFactory) (*CredentialsAwareClientFactory, error) {
	cacheDuration := 5 * time.Minute

	return &CredentialsAwareClientFactory{
		azureCredentialProvider: azureCredentialProvider,
		azureClientFactory:      azureClientFactory,
		cachedClients:           gocache.New(cacheDuration, 2*cacheDuration),
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
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "DeploymentsClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*resources.DeploymentsClient), nil
}

func (f *CredentialsAwareClientFactory) GetDnsRecordSetsClient(ctx context.Context, clusterID string) (*dns.RecordSetsClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "DnsRecordSetsClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*dns.RecordSetsClient), nil
}

func (f *CredentialsAwareClientFactory) GetGroupsClient(ctx context.Context, clusterID string) (*resources.GroupsClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "GroupsClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*resources.GroupsClient), nil
}

func (f *CredentialsAwareClientFactory) GetInterfacesClient(ctx context.Context, clusterID string) (*network.InterfacesClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "InterfacesClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*network.InterfacesClient), nil
}

func (f *CredentialsAwareClientFactory) GetNatGatewaysClient(ctx context.Context, clusterID string) (*network.NatGatewaysClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "NatGatewaysClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*network.NatGatewaysClient), nil
}

func (f *CredentialsAwareClientFactory) GetNetworkSecurityGroupsClient(ctx context.Context, clusterID string) (*network.SecurityGroupsClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "SecurityGroupsClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*network.SecurityGroupsClient), nil
}

func (f *CredentialsAwareClientFactory) GetPublicIpAddressesClient(ctx context.Context, clusterID string) (*network.PublicIPAddressesClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "PublicIPAddressesClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*network.PublicIPAddressesClient), nil
}

func (f *CredentialsAwareClientFactory) GetResourceSkusClient(ctx context.Context, clusterID string) (*compute.ResourceSkusClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "ResourceSkusClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*compute.ResourceSkusClient), nil
}

func (f *CredentialsAwareClientFactory) GetStorageAccountsClient(ctx context.Context, clusterID string) (*storage.AccountsClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "AccountsClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*storage.AccountsClient), nil
}

func (f *CredentialsAwareClientFactory) GetSubnetsClient(ctx context.Context, clusterID string) (*network.SubnetsClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "SubnetsClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*network.SubnetsClient), nil
}

func (f *CredentialsAwareClientFactory) GetVirtualMachineScaleSetsClient(ctx context.Context, clusterID string) (*compute.VirtualMachineScaleSetsClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "VirtualMachineScaleSetsClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*compute.VirtualMachineScaleSetsClient), nil
}

func (f *CredentialsAwareClientFactory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, clusterID string) (*compute.VirtualMachineScaleSetVMsClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "VirtualMachineScaleSetVMsClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*compute.VirtualMachineScaleSetVMsClient), nil
}

func (f *CredentialsAwareClientFactory) GetVirtualNetworkGatewayConnectionsClient(ctx context.Context, clusterID string) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "VirtualNetworkGatewayConnectionsClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*network.VirtualNetworkGatewayConnectionsClient), nil
}

func (f *CredentialsAwareClientFactory) GetVirtualNetworkGatewaysClient(ctx context.Context, clusterID string) (*network.VirtualNetworkGatewaysClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "VirtualNetworkGatewaysClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*network.VirtualNetworkGatewaysClient), nil
}

func (f *CredentialsAwareClientFactory) GetVirtualNetworksClient(ctx context.Context, clusterID string) (*network.VirtualNetworksClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "VirtualNetworksClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*network.VirtualNetworksClient), nil
}

func (f *CredentialsAwareClientFactory) GetZonesClient(ctx context.Context, clusterID string) (*dns.ZonesClient, error) {
	initFunc := func(ctx context.Context, clusterID string) (interface{}, error) {
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

	client, err := f.cacheLookup(ctx, "ZonesClient", clusterID, initFunc)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return client.(*dns.ZonesClient), nil
}

func (f *CredentialsAwareClientFactory) cacheLookup(ctx context.Context, clientType string, clusterID string, initFunc func(context.Context, string) (interface{}, error)) (interface{}, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	clientKey := fmt.Sprintf("%s-%s", clusterID, clientType)

	var err error
	var client interface{}
	if cachedClient, ok := f.cachedClients.Get(clientKey); ok {
		// client found, it will be refreshed in cache
		client = cachedClient
	} else {
		client, err = initFunc(ctx, clusterID)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		f.cachedClients.SetDefault(clientKey, client)
	}

	return client, nil
}
