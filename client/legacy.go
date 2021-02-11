package client

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	apiextensionslabels "github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// GetLegacyCredentialSecret is used by the azureclusteridentity handler to migrate from the legacy secret system to AzureClusterIdentity.
func (f *OrganizationFactory) GetLegacyCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	f.logger.Debugf(ctx, "finding credential secret")

	var err error
	var credentialSecret *v1alpha1.CredentialSecret

	credentialSecret, err = f.getLegacyOrganizationCredentialSecret(ctx, objectMeta)
	if IsCredentialsNotFoundError(err) {
		credentialSecret, err = f.getLegacyCredentialSecret(ctx, objectMeta)
		if IsCredentialsNotFoundError(err) {
			f.logger.Debugf(ctx, "did not find credential secret, using default '%s/%s'", credentialDefaultNamespace, credentialDefaultName)
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

// getLegacyOrganizationCredentialSecret tries to find a Secret labeled with the organization ID.
func (f *OrganizationFactory) getLegacyOrganizationCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	f.logger.Debugf(ctx, "try in namespace %#q filtering by organization %#q", objectMeta.Namespace, key.OrganizationID(&objectMeta))
	secretList := &corev1.SecretList{}
	{
		err := f.ctrlClient.List(
			ctx,
			secretList,
			client.InNamespace(objectMeta.Namespace),
			client.MatchingLabels{
				label.App:                        "credentiald",
				apiextensionslabels.Organization: key.OrganizationID(&objectMeta),
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

	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: secret.Namespace,
		Name:      secret.Name,
	}

	f.logger.Debugf(ctx, "found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name)

	return credentialSecret, nil
}

// getLegacyCredentialSecret tries to find a Secret in the default credentials namespace but labeled with the organization name.
// This is needed while we migrate everything to the org namespace and org credentials are created in the org namespace instead of the default namespace.
func (f *OrganizationFactory) getLegacyCredentialSecret(ctx context.Context, objectMeta v1.ObjectMeta) (*v1alpha1.CredentialSecret, error) {
	f.logger.Debugf(ctx, "try in namespace %#q filtering by organization %#q", credentialDefaultNamespace, key.OrganizationID(&objectMeta))
	secretList := &corev1.SecretList{}
	{
		err := f.ctrlClient.List(
			ctx,
			secretList,
			client.InNamespace(credentialDefaultNamespace),
			client.MatchingLabels{
				label.App:                        "credentiald",
				apiextensionslabels.Organization: key.OrganizationID(&objectMeta),
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

	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: secret.Namespace,
		Name:      secret.Name,
	}

	f.logger.Debugf(ctx, "found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name)

	return credentialSecret, nil
}
