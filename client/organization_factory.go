package client

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	credentialDefaultNamespace = "giantswarm"
	credentialDefaultName      = "credential-default"
)

type Interface interface {
	GetLegacyCredentialSecret(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*v1alpha1.CredentialSecret, error)
	GetDeploymentsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*resources.DeploymentsClient, error)
	GetDisksClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.DisksClient, error)
	GetGroupsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*resources.GroupsClient, error)
	GetInterfacesClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*network.InterfacesClient, error)
	GetDNSRecordSetsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*dns.RecordSetsClient, error)
	GetVirtualMachineScaleSetsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.VirtualMachineScaleSetsClient, error)
	GetVirtualMachineScaleSetVMsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.VirtualMachineScaleSetVMsClient, error)
	GetVirtualNetworksClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*network.VirtualNetworksClient, error)
	GetSnapshotsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.SnapshotsClient, error)
	GetStorageAccountsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*storage.AccountsClient, error)
	GetSubnetsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*network.SubnetsClient, error)
	GetNatGatewaysClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*network.NatGatewaysClient, error)
	GetResourceSkusClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.ResourceSkusClient, error)
}

type OrganizationFactoryConfig struct {
	CtrlClient client.Client
	Factory    *Factory
	Logger     micrologger.Logger
}

type OrganizationFactory struct {
	ctrlClient client.Client
	factory    *Factory
	logger     micrologger.Logger
}

func NewOrganizationFactory(c OrganizationFactoryConfig) OrganizationFactory {
	return OrganizationFactory{
		factory:    c.Factory,
		logger:     c.Logger,
		ctrlClient: c.CtrlClient,
	}
}

func (f *OrganizationFactory) GetDeploymentsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*resources.DeploymentsClient, error) {
	return f.factory.GetDeploymentsClient(azureCluster)
}

func (f *OrganizationFactory) GetDisksClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.DisksClient, error) {
	return f.factory.GetDisksClient(azureCluster)
}

func (f *OrganizationFactory) GetGroupsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*resources.GroupsClient, error) {
	return f.factory.GetGroupsClient(azureCluster)
}

func (f *OrganizationFactory) GetInterfacesClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*network.InterfacesClient, error) {
	return f.factory.GetInterfacesClient(azureCluster)
}

func (f *OrganizationFactory) GetDNSRecordSetsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*dns.RecordSetsClient, error) {
	return f.factory.GetDNSRecordSetsClient(azureCluster)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.VirtualMachineScaleSetsClient, error) {
	return f.factory.GetVirtualMachineScaleSetsClient(azureCluster)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.VirtualMachineScaleSetVMsClient, error) {
	return f.factory.GetVirtualMachineScaleSetVMsClient(azureCluster)
}

func (f *OrganizationFactory) GetVirtualNetworksClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*network.VirtualNetworksClient, error) {
	return f.factory.GetVirtualNetworksClient(azureCluster)
}

func (f *OrganizationFactory) GetSnapshotsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.SnapshotsClient, error) {
	return f.factory.GetSnapshotsClient(azureCluster)
}

func (f *OrganizationFactory) GetStorageAccountsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*storage.AccountsClient, error) {
	return f.factory.GetStorageAccountsClient(azureCluster)
}

func (f *OrganizationFactory) GetSubnetsClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*network.SubnetsClient, error) {
	return f.factory.GetSubnetsClient(azureCluster)
}

func (f *OrganizationFactory) GetNatGatewaysClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*network.NatGatewaysClient, error) {
	return f.factory.GetNatGatewaysClient(azureCluster)
}

func (f *OrganizationFactory) GetResourceSkusClient(ctx context.Context, azureCluster v1alpha3.AzureCluster) (*compute.ResourceSkusClient, error) {
	return f.factory.GetResourceSkusClient(azureCluster)
}
