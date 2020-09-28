package client

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	credentialDefaultName = "credential-default"
)

type Interface interface {
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
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetDeploymentsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetDisksClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.DisksClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetDisksClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetGroupsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*resources.GroupsClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetGroupsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetInterfacesClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.InterfacesClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetInterfacesClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetDNSRecordSetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*dns.RecordSetsClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetDNSRecordSetsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.VirtualMachineScaleSetsClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualMachineScaleSetsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.VirtualMachineScaleSetVMsClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualMachineScaleSetVMsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualNetworksClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworksClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualNetworksClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetSnapshotsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.SnapshotsClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetSnapshotsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetStorageAccountsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*storage.AccountsClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetStorageAccountsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetSubnetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.SubnetsClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetSubnetsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetNatGatewaysClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.NatGatewaysClient, error) {
	credentialSecret, err := f.getCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetNatGatewaysClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) getCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	f.logger.LogCtx(ctx, "level", "debug", "message", "finding credential secret")

	secretList := &corev1.SecretList{}
	{
		err := f.ctrlClient.List(
			ctx,
			secretList,
			client.InNamespace(objectMeta.Namespace),
			client.MatchingLabels{
				label.App:          "credentiald",
				label.Organization: key.OrganizationID(&objectMeta),
			},
		)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	// We currently only support one credential secret per organization.
	// If there are more than one, return an error.
	if len(secretList.Items) > 1 {
		return nil, microerror.Mask(tooManyCredentialsError)
	}

	// If one credential secret is found, we use that.
	if len(secretList.Items) == 1 {
		secret := secretList.Items[0]

		credentialSecret := &v1alpha1.CredentialSecret{
			Namespace: secret.Namespace,
			Name:      secret.Name,
		}

		f.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name))

		return credentialSecret, nil
	}

	// If no credential secrets are found, we use the default.
	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: objectMeta.Namespace,
		Name:      credentialDefaultName,
	}

	f.logger.LogCtx(ctx, "level", "debug", "message", "did not find credential secret, using default secret")

	return credentialSecret, nil
}
