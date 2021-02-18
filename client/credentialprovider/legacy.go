package credentialprovider

import (
	"context"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	apiextensionslabels "github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
)

const (
	credentialDefaultNamespace = "giantswarm"
	credentialDefaultName      = "credential-default"
)

// GetLegacyCredentialSecret Tries to find the legacy GS credential secret for the given organization.
func (k *K8sSecretCredentialProvider) GetLegacyCredentialSecret(ctx context.Context, organizationID string) (*v1alpha1.CredentialSecret, error) {
	k.logger.Debugf(ctx, "finding credential secret")

	var err error
	var credentialSecret *v1alpha1.CredentialSecret

	credentialSecret, err = k.getOrganizationCredentialSecret(ctx, organizationID)
	if IsCredentialsNotFoundError(err) {
		k.logger.Debugf(ctx, "did not find credential secret, using default '%s/%s'", credentialDefaultNamespace, credentialDefaultName)
		return &v1alpha1.CredentialSecret{
			Namespace: credentialDefaultNamespace,
			Name:      credentialDefaultName,
		}, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return credentialSecret, nil
}

// getOrganizationCredentialSecret tries to find a Secret for the organization (BYOC).
func (k *K8sSecretCredentialProvider) getOrganizationCredentialSecret(ctx context.Context, organizationID string) (*v1alpha1.CredentialSecret, error) {
	k.logger.Debugf(ctx, "try in all namespaces filtering by organization %#q", organizationID)
	secretList := &corev1.SecretList{}
	{
		err := k.ctrlClient.List(
			ctx,
			secretList,
			client.MatchingLabels{
				label.App:                        "credentiald",
				apiextensionslabels.Organization: organizationID,
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

	k.logger.Debugf(ctx, "found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name)

	return credentialSecret, nil
}
