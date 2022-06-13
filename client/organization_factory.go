package client

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/authorization/mgmt/authorization"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/giantswarm/apiextensions/v6/pkg/apis/provider/v1alpha1"
	k8smetadatalabel "github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	credentialDefaultNamespace = "giantswarm"
	credentialLegacyNamespace  = "giantswarm"
	credentialDefaultName      = "credential-default" // nolint:gosec
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
	GetVnetPeeringsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworkPeeringsClient, error)
	GetVirtualNetworkGatewaysClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworkGatewaysClient, error)
	GetVirtualNetworkGatewayConnectionsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworkGatewayConnectionsClient, error)
	GetPublicIpAddressesClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.PublicIPAddressesClient, error)
	GetRoleAssignmentsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*authorization.RoleAssignmentsClient, error)
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
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetDeploymentsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetDisksClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.DisksClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetDisksClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetGroupsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*resources.GroupsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetGroupsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetInterfacesClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.InterfacesClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetInterfacesClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetDNSRecordSetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*dns.RecordSetsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetDNSRecordSetsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.VirtualMachineScaleSetsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualMachineScaleSetsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualMachineScaleSetVMsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.VirtualMachineScaleSetVMsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualMachineScaleSetVMsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualNetworksClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworksClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualNetworksClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetSnapshotsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.SnapshotsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetSnapshotsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetStorageAccountsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*storage.AccountsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetStorageAccountsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetSubnetsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.SubnetsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetSubnetsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetNatGatewaysClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.NatGatewaysClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetNatGatewaysClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetResourceSkusClient(ctx context.Context, objectMeta v1.ObjectMeta) (*compute.ResourceSkusClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetResourceSkusClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVnetPeeringsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworkPeeringsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualNetworkPeeringsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualNetworkGatewaysClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworkGatewaysClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualNetworkGatewaysClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetVirtualNetworkGatewayConnectionsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.VirtualNetworkGatewayConnectionsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetVirtualNetworkGatewayConnectionsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetPublicIpAddressesClient(ctx context.Context, objectMeta v1.ObjectMeta) (*network.PublicIPAddressesClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetPublicIPAddressesClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetRoleAssignmentsClient(ctx context.Context, objectMeta v1.ObjectMeta) (*authorization.RoleAssignmentsClient, error) {
	credentialSecret, err := f.GetCredentialSecret(ctx, objectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f.factory.GetRoleAssignmentsClient(credentialSecret.Namespace, credentialSecret.Name)
}

func (f *OrganizationFactory) GetCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	f.logger.Debugf(ctx, "finding credential secret")

	var err error
	var credentialSecret *v1alpha1.CredentialSecret

	credentialSecret, err = f.getOrganizationCredentialSecret(ctx, objectMeta)
	if IsCredentialsNotFoundError(err) {
		// TODO remove once all credentials are migrated to the org namespace.
		credentialSecret, err = f.tryMigrateLegacyCredentialSecret(ctx, objectMeta)
		if IsCredentialsNotFoundError(err) {
			f.logger.Debugf(ctx, "did not find credentials in the org nor in the legacy namespaces. Using default credentials %s/%s", credentialDefaultNamespace, credentialDefaultName)
			return &v1alpha1.CredentialSecret{
				Namespace: credentialDefaultNamespace,
				Name:      credentialDefaultName,
			}, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return credentialSecret, nil
}

// getOrganizationCredentialSecret tries to find a Secret in the organization namespace.
func (f *OrganizationFactory) getOrganizationCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	ns := key.OrganizationNamespace(&objectMeta)
	name := key.OrganizationID(&objectMeta)
	f.logger.Debugf(ctx, "try in namespace %#q filtering by organization %#q", ns, name)
	secretList := &corev1.SecretList{}
	{
		err := f.ctrlClient.List(
			ctx,
			secretList,
			client.InNamespace(ns),
			client.MatchingLabels{
				label.App:                     "credentiald",
				k8smetadatalabel.Organization: name,
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

	if len(secretList.Items) < 1 {
		return nil, microerror.Mask(credentialsNotFoundError)
	}

	// If one credential secret is found, we use that.
	secret := secretList.Items[0]

	if secret.Name == credentialDefaultName {
		// Some default credentials might have the 'giantswarm' organization label.
		// We want to avoid using the default credentials secret as org credentials
		return nil, microerror.Mask(credentialsNotFoundError)
	}

	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: secret.Namespace,
		Name:      secret.Name,
	}

	f.logger.Debugf(ctx, "found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name)

	return credentialSecret, nil
}

// tryMigrateLegacyCredentialSecret tries to find a Secret in the default credentials namespace but labeled with the organization name.
// This is the legacy location where credentials used to be stored.
// In case such credential is found, an attempt to move it to the org namespace is made.
func (f *OrganizationFactory) tryMigrateLegacyCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	f.logger.Debugf(ctx, "try in namespace %#q filtering by organization %#q", credentialLegacyNamespace, key.OrganizationID(&objectMeta))
	secretList := &corev1.SecretList{}
	{
		err := f.ctrlClient.List(
			ctx,
			secretList,
			client.InNamespace(credentialDefaultNamespace),
			client.MatchingLabels{
				label.App:                     "credentiald",
				k8smetadatalabel.Organization: key.OrganizationID(&objectMeta),
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

	if len(secretList.Items) < 1 {
		return nil, microerror.Mask(credentialsNotFoundError)
	}

	// One secret was found in the legacy location, migrate it.
	// If we reached this point, we are already sure there is no secret in the org namespace for this org.
	secret := secretList.Items[0]

	if secret.Name == credentialDefaultName {
		// Some default credentials might have the 'giantswarm' organization label.
		// We want to avoid using the default credentials secret as org credentials
		return nil, microerror.Mask(credentialsNotFoundError)
	}

	f.logger.Debugf(ctx, "migrating secret from legacy namespace %q to org namespace %q", credentialLegacyNamespace, key.OrganizationNamespace(&objectMeta))

	{
		newSecret := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:        secret.Name,
				Namespace:   key.OrganizationNamespace(&objectMeta),
				Labels:      secret.Labels,
				Annotations: secret.Annotations,
			},
			Data: secret.Data,
			Type: secret.Type,
		}

		err := f.ctrlClient.Create(ctx, newSecret)
		if err != nil {
			f.logger.Debugf(ctx, "error migrating secret from legacy namespace %q to org namespace %q", credentialLegacyNamespace, key.OrganizationNamespace(&objectMeta))
			return nil, microerror.Mask(err)
		}

		// Delete old secret object.
		err = f.ctrlClient.Delete(ctx, &secret)
		if err != nil {
			f.logger.Debugf(ctx, "error cleaning up legacy secret %q/%q after successful migration to org namespace", credentialLegacyNamespace, secret.Name)
			return nil, microerror.Mask(err)
		}

		secret = *newSecret
	}

	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: secret.Namespace,
		Name:      secret.Name,
	}

	f.logger.Debugf(ctx, "found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name)

	return credentialSecret, nil
}
