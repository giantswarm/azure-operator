package azureclusteridentity

import (
	"context"
	"reflect"

	apiextensionslabels "github.com/giantswarm/apiextensions/v6/pkg/label"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v8/pkg/project"
	"github.com/giantswarm/azure-operator/v8/service/controller/key"
)

const (
	secretDataFieldName = "clientSecret"

	legacySecretClientIDFieldName       = "azure.azureoperator.clientid"
	legacySecretClientSecretFieldName   = "azure.azureoperator.clientsecret"
	legacySecretSubscriptionIDFieldName = "azure.azureoperator.subscriptionid"
	legacySecretTenantIDFieldName       = "azure.azureoperator.tenantid"
)

// EnsureCreated ensures there is an AzureClusterIdentity CR and a related Secret
// with the same contents as the Giant Swarm credential secret.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	var err error
	azureCluster, err := key.ToAzureCluster(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Retrieve the legacy secret related to the organization this AzureCluster belongs to.
	var legacySecret corev1.Secret
	{
		credentialSecret, err := r.azureClientsFactory.GetCredentialSecret(ctx, azureCluster.ObjectMeta)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.ctrlClient.Get(ctx, client.ObjectKey{Name: credentialSecret.Name, Namespace: credentialSecret.Namespace}, &legacySecret)
		if apierrors.IsNotFound(err) {
			// Legacy secret does not exist, we can do nothing but hope the IdentityRef is set.
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	if azureCluster.Spec.IdentityRef == nil {
		r.logger.Debugf(ctx, "AzureCluster %q has no IdentityRef set, setting it from Secret %q in namespace %q", azureCluster.Name, legacySecret.Name, legacySecret.Namespace)

		err = r.ensureNewSecret(ctx, &azureCluster, legacySecret)
		if err != nil {
			return microerror.Mask(err)
		}

		identity, err := r.ensureAzureClusterIdentity(ctx, &azureCluster, legacySecret)
		if err != nil {
			return microerror.Mask(err)
		}

		azureCluster.Spec.IdentityRef = &corev1.ObjectReference{
			Kind:      identity.Kind,
			Name:      identity.Name,
			Namespace: identity.Namespace,
		}
		err = r.ctrlClient.Update(ctx, &azureCluster)
		if apierrors.IsConflict(err) {
			r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "Set IdentityRef for AzureCluster %q", azureCluster.Name)
	} else {
		// Ensure AzureClusterIdentity is up to date.
		azureClusterIdentity := capz.AzureClusterIdentity{}
		err := r.ctrlClient.Get(ctx, client.ObjectKey{Name: azureCluster.Spec.IdentityRef.Name, Namespace: azureCluster.Spec.IdentityRef.Namespace}, &azureClusterIdentity)
		if apierrors.IsNotFound(err) {
			// identityRef is set to a non-existing object. Empty the reference.
			r.logger.Debugf(ctx, "AzureClusterIdentity %s/%s was not found for AzureCluster %q", azureCluster.Spec.IdentityRef.Namespace, azureCluster.Spec.IdentityRef.Name, azureCluster.Name)
			azureCluster.Spec.IdentityRef = nil
			err = r.ctrlClient.Update(ctx, &azureCluster)
			if err != nil {
				return microerror.Mask(err)
			}
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}

		// If AzureClusterIdentity was not created by azure-operator, ignore it
		managedBy := azureClusterIdentity.Labels[apiextensionslabels.ManagedBy]
		if managedBy != project.Name() {
			return nil
		}

		newSecret := corev1.Secret{}
		err = r.ctrlClient.Get(ctx, client.ObjectKey{Name: azureClusterIdentity.Spec.ClientSecret.Name, Namespace: azureClusterIdentity.Spec.ClientSecret.Namespace}, &newSecret)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.ensureNewSecretUpdated(ctx, legacySecret, newSecret)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.ensureAzureClusterIdentityUpdated(ctx, &azureCluster, legacySecret, azureClusterIdentity)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Check if the AzureCluster CR has a SubscriptionID set.
	if azureCluster.Spec.SubscriptionID == "" {
		r.logger.Debugf(ctx, "AzureCluster doesn't have a Subscription ID set.")
		azureCluster.Spec.SubscriptionID = string(legacySecret.Data[legacySecretSubscriptionIDFieldName])
		err = r.ctrlClient.Update(ctx, &azureCluster)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Set Subscription ID %q in AzureCluster", string(legacySecret.Data[legacySecretSubscriptionIDFieldName]))
	}

	return nil
}

func (r *Resource) ensureNewSecret(ctx context.Context, azureCluster *capz.AzureCluster, legacySecret corev1.Secret) error {
	newName := newSecretName(legacySecret)

	r.logger.Debugf(ctx, "Looking for Secret %q in namespace %q", newName, azureCluster.Namespace)

	clientSecret := string(legacySecret.Data[legacySecretClientSecretFieldName])

	existing := &corev1.Secret{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureCluster.Namespace, Name: newName}, existing)
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "Secret %q wasn't found in namespace %q, creating it", newName, azureCluster.Namespace)

		// We need to create the secret.
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      newName,
				Namespace: azureCluster.Namespace,
				Labels: map[string]string{
					apiextensionslabels.ManagedBy:    project.Name(),
					apiextensionslabels.Organization: legacySecret.GetLabels()[apiextensionslabels.Organization],
				},
			},
			StringData: map[string]string{
				secretDataFieldName: clientSecret,
			},
		}

		err := r.ctrlClient.Create(ctx, newSecret)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "Secret %q created in namespace %q", newName, azureCluster.Namespace)

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Secret %q found in namespace %q", newName, azureCluster.Namespace)

	return nil
}

func (r *Resource) ensureNewSecretUpdated(ctx context.Context, legacySecret corev1.Secret, newSecret corev1.Secret) error {
	clientSecret := string(legacySecret.Data[legacySecretClientSecretFieldName])

	currentClientSecret := string(newSecret.Data[secretDataFieldName])
	if currentClientSecret != clientSecret {
		r.logger.Debugf(ctx, "Secret %q is outdated, updating", newSecret.Name)

		newSecret.StringData = map[string]string{
			secretDataFieldName: clientSecret,
		}
		err := r.ctrlClient.Update(ctx, &newSecret)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Secret %q updated successfully", newSecret.Name)
		return nil
	}

	r.logger.Debugf(ctx, "Secret %q is up to date", newSecret.Name)

	return nil
}

func (r *Resource) ensureAzureClusterIdentity(ctx context.Context, azureCluster *capz.AzureCluster, legacySecret corev1.Secret) (*capz.AzureClusterIdentity, error) {
	newName := newSecretName(legacySecret)
	desiredSpec := getAzureClusterIdentitySpec(azureCluster, legacySecret)

	r.logger.Debugf(ctx, "Looking for AzureClusterIdentity %q in namespace %q", newName, azureCluster.Namespace)

	existing := &capz.AzureClusterIdentity{}
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: azureCluster.Namespace, Name: newName}, existing)
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "AzureClusterIdentity %q wasn't found in namespace %q, creating it", newName, azureCluster.Namespace)

		// We need to create the AzureClusterIdentity.
		aci := &capz.AzureClusterIdentity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      newName,
				Namespace: azureCluster.Namespace,
				Labels: map[string]string{
					apiextensionslabels.ManagedBy:    project.Name(),
					apiextensionslabels.Organization: legacySecret.GetLabels()[apiextensionslabels.Organization],
				},
			},
			Spec: desiredSpec,
		}

		err := r.ctrlClient.Create(ctx, aci)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "AzureClusterIdentity %q created in namespace %q", newName, azureCluster.Namespace)

		return aci, nil

	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "AzureClusterIdentity %q found in namespace %q", newName, azureCluster.Namespace)

	return existing, nil
}

func (r *Resource) ensureAzureClusterIdentityUpdated(ctx context.Context, azureCluster *capz.AzureCluster, legacySecret corev1.Secret, existing capz.AzureClusterIdentity) error {
	newName := newSecretName(legacySecret)
	desiredSpec := getAzureClusterIdentitySpec(azureCluster, legacySecret)

	if !reflect.DeepEqual(existing.Spec, desiredSpec) {
		r.logger.Debugf(ctx, "AzureClusterIdentity %q is outdated, updating", newName)
		existing.Spec = desiredSpec

		err := r.ctrlClient.Update(ctx, &existing)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Debugf(ctx, "AzureClusterIdentity %q updated successfully", newName)

		return nil
	}

	r.logger.Debugf(ctx, "AzureClusterIdentity %q is up to date", newName)

	return nil
}

func getAzureClusterIdentitySpec(azureCluster *capz.AzureCluster, legacySecret corev1.Secret) capz.AzureClusterIdentitySpec {
	newName := newSecretName(legacySecret)
	newNamespace := azureCluster.Namespace

	clientID := string(legacySecret.Data[legacySecretClientIDFieldName])
	tenantID := string(legacySecret.Data[legacySecretTenantIDFieldName])

	return capz.AzureClusterIdentitySpec{
		Type:     capz.ServicePrincipal,
		ClientID: clientID,
		ClientSecret: corev1.SecretReference{
			Name:      newName,
			Namespace: newNamespace,
		},
		TenantID:          tenantID,
		AllowedNamespaces: nil,
	}
}
