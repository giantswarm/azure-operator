package client

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	apiextensionslabels "github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	credentialDefaultNamespace = "giantswarm"
	credentialDefaultName      = "credential-default"
)

type Interface interface {
	GetLegacyCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error)
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

func (f *OrganizationFactory) GetDeploymentsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*resources.DeploymentsClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetDeploymentsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetDisksClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.DisksClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetDisksClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetGroupsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*resources.GroupsClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetGroupsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetInterfacesClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.InterfacesClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetInterfacesClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetDNSRecordSetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*dns.RecordSetsClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetDNSRecordSetsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.VirtualMachineScaleSetsClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualMachineScaleSetsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.VirtualMachineScaleSetVMsClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualMachineScaleSetVMsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualNetworksClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworksClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualNetworksClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetSnapshotsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.SnapshotsClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetSnapshotsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetStorageAccountsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*storage.AccountsClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetStorageAccountsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetSubnetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.SubnetsClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetSubnetsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetNatGatewaysClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.NatGatewaysClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetNatGatewaysClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetResourceSkusClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.ResourceSkusClient, error) {
	credentialSecret, err := f.getAzureClusterIdentity(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetResourceSkusClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) getAzureClusterIdentity(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha3.AzureClusterIdentity, error) {
	f.logger.Debugf(ctx, "finding credential secret")

	azureClusterIdentity, err := f.getOrganizationCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClusterIdentity, nil
}

// getOrganizationCredentialSecret tries to find an AzureClusterIdentity labeled with the organization ID.
func (f *OrganizationFactory) getOrganizationCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha3.AzureClusterIdentity, error) {
	f.logger.Debugf(ctx, "try in all namespaces filtering by organization %#q", objectMeta.Namespace, key.OrganizationID(&objectMeta))
	azureClusterIdentityList := &v1alpha3.AzureClusterIdentityList{}
	{
		err := f.ctrlClient.List(
			ctx,
			azureClusterIdentityList,
			client.MatchingLabels{
				apiextensionslabels.Organization: key.OrganizationID(&objectMeta),
			},
		)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// We currently only support one credential azureClusterIdentity per organization.
	// If there are more than one, return an error.
	if len(azureClusterIdentityList.Items) > 1 {
		return nil, microerror.Mask(tooManyCredentialsError)
	}

	if len(azureClusterIdentityList.Items) < 1 {
		return nil, microerror.Mask(credentialsNotFoundError)
	}

	azureClusterIdentity := azureClusterIdentityList.Items[0]

	f.logger.Debugf(ctx, "found azureClusterIdentity %s/%s", azureClusterIdentity.Namespace, azureClusterIdentity.Name)

	// If one credential azureClusterIdentity is found, we use that.
	return &azureClusterIdentity, nil
}
