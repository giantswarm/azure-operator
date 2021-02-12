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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	credentialDefaultNamespace = "giantswarm"
	credentialDefaultName      = "credential-default"
)

type Interface interface {
	GetCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error)
	GetDeploymentsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*resources.DeploymentsClient, error)
	GetDisksClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.DisksClient, error)
	GetGroupsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*resources.GroupsClient, error)
	GetInterfacesClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.InterfacesClient, error)
	GetDNSRecordSetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*dns.RecordSetsClient, error)
	GetVirtualMachineScaleSetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.VirtualMachineScaleSetsClient, error)
	GetVirtualMachineScaleSetVMsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.VirtualMachineScaleSetVMsClient, error)
	GetVirtualNetworksClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworksClient, error)
	GetSnapshotsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.SnapshotsClient, error)
	GetStorageAccountsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*storage.AccountsClient, error)
	GetSubnetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.SubnetsClient, error)
	GetNatGatewaysClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.NatGatewaysClient, error)
	GetResourceSkusClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.ResourceSkusClient, error)
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

func (f *OrganizationFactory) GetDeploymentsClient(ctx context.Context, clusterID string) (*resources.DeploymentsClient, error) {
	return f.factory.GetDeploymentsClient(clusterID)
}

func (f *OrganizationFactory) GetDisksClient(ctx context.Context, clusterID string) (*compute.DisksClient, error) {
	return f.factory.GetDisksClient(clusterID)
}

func (f *OrganizationFactory) GetGroupsClient(ctx context.Context, clusterID string) (*resources.GroupsClient, error) {
	return f.factory.GetGroupsClient(clusterID)
}

func (f *OrganizationFactory) GetInterfacesClient(ctx context.Context, clusterID string) (*network.InterfacesClient, error) {
	return f.factory.GetInterfacesClient(clusterID)
}

func (f *OrganizationFactory) GetDNSRecordSetsClient(ctx context.Context, clusterID string) (*dns.RecordSetsClient, error) {
	return f.factory.GetDNSRecordSetsClient(clusterID)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetsClient(ctx context.Context, clusterID string) (*compute.VirtualMachineScaleSetsClient, error) {
	return f.factory.GetVirtualMachineScaleSetsClient(clusterID)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, clusterID string) (*compute.VirtualMachineScaleSetVMsClient, error) {
	return f.factory.GetVirtualMachineScaleSetVMsClient(clusterID)
}

func (f *OrganizationFactory) GetVirtualNetworksClient(ctx context.Context, clusterID string) (*network.VirtualNetworksClient, error) {
	return f.factory.GetVirtualNetworksClient(clusterID)
}

func (f *OrganizationFactory) GetSnapshotsClient(ctx context.Context, clusterID string) (*compute.SnapshotsClient, error) {
	return f.factory.GetSnapshotsClient(clusterID)
}

func (f *OrganizationFactory) GetStorageAccountsClient(ctx context.Context, clusterID string) (*storage.AccountsClient, error) {
	return f.factory.GetStorageAccountsClient(clusterID)
}

func (f *OrganizationFactory) GetSubnetsClient(ctx context.Context, clusterID string) (*network.SubnetsClient, error) {
	return f.factory.GetSubnetsClient(clusterID)
}

func (f *OrganizationFactory) GetNatGatewaysClient(ctx context.Context, clusterID string) (*network.NatGatewaysClient, error) {
	return f.factory.GetNatGatewaysClient(clusterID)
}

func (f *OrganizationFactory) GetResourceSkusClient(ctx context.Context, clusterID string) (*compute.ResourceSkusClient, error) {
	return f.factory.GetResourceSkusClient(clusterID)
}
