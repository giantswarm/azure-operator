package azureconfig

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	credentialDefaultName = "credential-default"
	credentialNamespace   = "giantswarm"
)

func (r *Resource) getCredentialSecret(ctx context.Context, cluster capiv1alpha3.Cluster) (*v1alpha1.CredentialSecret, error) {
	r.logger.LogCtx(ctx, "level", "debug", "message", "finding credential secret")

	secretList := &corev1.SecretList{}
	{
		err := r.ctrlClient.List(
			ctx,
			secretList,
			client.InNamespace(credentialNamespace),
			client.MatchingLabels{
				label.App:          "credentiald",
				label.Organization: key.OrganizationID(&cluster),
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

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("found credential secret %s/%s", credentialSecret.Namespace, credentialSecret.Name))

		return credentialSecret, nil
	}

	// If no credential secrets are found, we use the default.
	credentialSecret := &v1alpha1.CredentialSecret{
		Namespace: credentialNamespace,
		Name:      credentialDefaultName,
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "did not find credential secret, using default secret")

	return credentialSecret, nil
}
