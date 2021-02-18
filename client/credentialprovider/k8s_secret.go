package credentialprovider

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/client/factory"
)

const (
	clientSecretKey = "clientSecret"
)

type K8sSecretCredentialProviderConfig struct {
	CtrlClient ctrl.Client
	Logger     micrologger.Logger

	TenantID string
}

type K8sSecretCredentialProvider struct {
	ctrlClient ctrl.Client
	logger     micrologger.Logger

	tenantID string
}

func NewK8sSecretCredentialProvider(config K8sSecretCredentialProviderConfig) (*K8sSecretCredentialProvider, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.TenantID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TenantID must not be empty", config)
	}

	return &K8sSecretCredentialProvider{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
		tenantID:   config.TenantID,
	}, nil
}

func (k *K8sSecretCredentialProvider) GetAzureClientCredentialsConfig(ctx context.Context, clusterID string) (*factory.AzureClientCredentialsConfig, error) {
	azureClusters := &v1alpha3.AzureClusterList{}
	err := k.ctrlClient.List(ctx, azureClusters, ctrl.MatchingLabels{label.Cluster: clusterID})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if len(azureClusters.Items) != 1 {
		return nil, microerror.Maskf(azureClusterNotFoundError, "Expected 1 AzureCluster with label %s = %q, %d found", label.Cluster, clusterID, len(azureClusters.Items))
	}

	azureCluster := azureClusters.Items[0]

	if azureCluster.Spec.SubscriptionID == "" {
		return nil, microerror.Maskf(subscriptionIDNotSetError, "AzureCluster %s/%s didn't have the SubscriptionID field set", azureCluster.Namespace, azureCluster.Name)
	}

	if azureCluster.Spec.IdentityRef == nil {
		return nil, microerror.Maskf(identityRefNotSetError, "AzureCluster %s/%s didn't have the IdentityRef field set", azureCluster.Namespace, azureCluster.Name)
	}

	azureClusterIdentity := &v1alpha3.AzureClusterIdentity{}
	err = k.ctrlClient.Get(ctx, ctrl.ObjectKey{Namespace: azureCluster.Spec.IdentityRef.Namespace, Name: azureCluster.Spec.IdentityRef.Name}, azureClusterIdentity)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	secret := &v1.Secret{}
	err = k.ctrlClient.Get(ctx, ctrl.ObjectKey{Namespace: azureClusterIdentity.Spec.ClientSecret.Namespace, Name: azureClusterIdentity.Spec.ClientSecret.Name}, secret)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	subscriptionID := azureCluster.Spec.SubscriptionID
	clientID := azureClusterIdentity.Spec.ClientID
	subscriptionTenantID := azureClusterIdentity.Spec.TenantID
	mcSubscriptionTenantID := k.tenantID

	clientSecret, err := valueFromSecret(secret, clientSecretKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if mcSubscriptionTenantID != "" && subscriptionTenantID != mcSubscriptionTenantID {
		credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, subscriptionTenantID)
		credentials.AuxTenants = append(credentials.AuxTenants, mcSubscriptionTenantID)
		return &factory.AzureClientCredentialsConfig{
			ClientCredentialsConfig: credentials,
			SubscriptionID:          subscriptionID,
		}, nil
	}

	credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, subscriptionTenantID)

	if subscriptionTenantID != k.tenantID {
		auxTenants := []string{
			k.tenantID,
		}
		credentials.AuxTenants = auxTenants
	}

	return &factory.AzureClientCredentialsConfig{
		ClientCredentialsConfig: credentials,
		SubscriptionID:          subscriptionID,
	}, nil
}

func valueFromSecret(secret *v1.Secret, key string) (string, error) {
	v, ok := secret.Data[key]
	if !ok {
		return "", microerror.Maskf(missingValueError, key)
	}

	return string(v), nil
}
