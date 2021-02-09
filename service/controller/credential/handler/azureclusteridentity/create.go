package azureclusteridentity

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	secretDataFieldName = "clientSecret"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	legacySecret, err := key.ToSecret(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Found secret %q in namespace %q", legacySecret.Name, legacySecret.Namespace)

	err = r.ensureNewSecret(ctx, legacySecret)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureAzureClusterIdentity(ctx, legacySecret)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureNewSecret(ctx context.Context, legacySecret corev1.Secret) error {
	newName := newSecretName(legacySecret)
	newNamespace := newSecretNamespace(legacySecret)

	r.logger.Debugf(ctx, "Looking for Secret %q in namespace %q", newName, newNamespace)

	clientSecret := string(legacySecret.Data["azure.azureoperator.clientsecret"])

	existing := &corev1.Secret{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: newNamespace, Name: newName}, existing)
	if errors.IsNotFound(err) {
		r.logger.Debugf(ctx, "Secret %q wasn't found in namespace %q, creating it", newName, newNamespace)

		// We need to create the secret.
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        newName,
				Namespace:   newNamespace,
				Labels:      nil,
				Annotations: nil,
			},
			StringData: map[string]string{
				secretDataFieldName: clientSecret,
			},
		}

		err := r.ctrlClient.Create(ctx, newSecret)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "Secret %q created in namespace %q", newName, newNamespace)

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Secret %q found in namespace %q", newName, newNamespace)

	currentClientSecret := string(existing.Data[secretDataFieldName])
	if currentClientSecret != clientSecret {
		r.logger.Debugf(ctx, "Secret %q is outdated, updating", newName)

		existing.StringData = map[string]string{
			secretDataFieldName: clientSecret,
		}
		err := r.ctrlClient.Update(ctx, existing)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Secret %q updated successfully", newName)
		return nil
	}

	r.logger.Debugf(ctx, "Secret %q is up to date", newName)

	return nil
}

func (r *Resource) ensureAzureClusterIdentity(ctx context.Context, legacySecret corev1.Secret) error {
	newName := newSecretName(legacySecret)
	newNamespace := newSecretNamespace(legacySecret)

	clientID := string(legacySecret.Data["azure.azureoperator.clientid"])
	tenantID := string(legacySecret.Data["azure.azureoperator.tenantid"])

	desiredSpec := v1alpha3.AzureClusterIdentitySpec{
		Type:     v1alpha3.ServicePrincipal,
		ClientID: clientID,
		ClientSecret: corev1.SecretReference{
			Name:      newName,
			Namespace: newNamespace,
		},
		TenantID:          tenantID,
		AllowedNamespaces: make([]string, 0),
	}

	r.logger.Debugf(ctx, "Looking for AzureClusterIdentity %q in namespace %q", newName, newNamespace)

	existing := &v1alpha3.AzureClusterIdentity{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: newNamespace, Name: newName}, existing)
	if errors.IsNotFound(err) {
		r.logger.Debugf(ctx, "AzureClusterIdentity %q wasn't found in namespace %q, creating it", newName, newNamespace)

		// We need to create the AzureClusterIdentity.
		aci := &v1alpha3.AzureClusterIdentity{
			ObjectMeta: metav1.ObjectMeta{
				Name:        newName,
				Namespace:   newNamespace,
				Labels:      nil,
				Annotations: nil,
			},
			Spec: desiredSpec,
		}

		err := r.ctrlClient.Create(ctx, aci)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "AzureClusterIdentity %q created in namespace %q", newName, newNamespace)

		return nil

	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "AzureClusterIdentity %q found in namespace %q", newName, newNamespace)

	if !reflect.DeepEqual(existing.Spec, desiredSpec) {
		r.logger.Debugf(ctx, "AzureClusterIdentity %q is outdated, updating", newName)
		existing.Spec = desiredSpec

		err = r.ctrlClient.Update(ctx, existing)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "AzureClusterIdentity %q updated successfully", newName)

		return nil
	}

	r.logger.Debugf(ctx, "AzureClusterIdentity %q is up to date", newName)

	return nil
}

func newSecretName(legacySecret corev1.Secret) string {
	name := strings.TrimPrefix(legacySecret.Name, "credential-")

	return fmt.Sprintf("org-credential-%s", name)
}

func newSecretNamespace(legacySecret corev1.Secret) string {
	return legacySecret.Namespace
}
