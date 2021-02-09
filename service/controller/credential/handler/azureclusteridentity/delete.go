package azureclusteridentity

import (
	"context"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	legacySecret, err := key.ToSecret(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureAzureClusterIdentityDeleted(ctx, legacySecret)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureSecretDeleted(ctx, legacySecret)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureAzureClusterIdentityDeleted(ctx context.Context, legacySecret corev1.Secret) error {
	newName := newSecretName(legacySecret)
	newNamespace := newSecretNamespace(legacySecret)

	existing := &v1alpha3.AzureClusterIdentity{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: newNamespace, Name: newName}, existing)
	if errors.IsNotFound(err) {
		// This is the desired state.
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Deleting AzureClusterIdentity %q from namespace %q", newName, newNamespace)
	err = r.ctrlClient.Delete(ctx, existing)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Deleted AzureClusterIdentity %q from namespace %q", newName, newNamespace)

	return nil
}

func (r *Resource) ensureSecretDeleted(ctx context.Context, legacySecret corev1.Secret) error {
	newName := newSecretName(legacySecret)
	newNamespace := newSecretNamespace(legacySecret)

	existing := &corev1.Secret{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: newNamespace, Name: newName}, existing)
	if errors.IsNotFound(err) {
		// This is the desired state.
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Deleting secret %q from namespace %q", newName, newNamespace)
	err = r.ctrlClient.Delete(ctx, existing)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Deleted secret %q from namespace %q", newName, newNamespace)

	return nil
}
