package credentialprovider

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clientSecretKey = "clientSecret"
)

type K8sSecretCredentialProviderConfig struct {
	CtrlClient ctrl.Client
	Logger     micrologger.Logger

	MCTenantID string
}

type K8sSecretCredentialProvider struct {
	ctrlClient ctrl.Client
	logger     micrologger.Logger

	mcTenantID string
}

func NewK8sSecretCredentialProvider(config K8sSecretCredentialProviderConfig) (*K8sSecretCredentialProvider, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.MCTenantID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.MCTenantID must not be empty", config)
	}

	return &K8sSecretCredentialProvider{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
		mcTenantID: config.MCTenantID,
	}, nil
}

func (k *K8sSecretCredentialProvider) GetAzureClientCredentialsConfig(ctx context.Context, clusterID string) (*AzureClientCredentialsConfig, error) {
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

	clientSecret, err := valueFromSecret(secret, clientSecretKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	credentials := auth.NewClientCredentialsConfig(clientID, clientSecret, subscriptionTenantID)

	if k.mcTenantID != "" && subscriptionTenantID != k.mcTenantID {
		oriAuxTenants := credentials.AuxTenants
		credentials.AuxTenants = append(oriAuxTenants, k.mcTenantID)

		// Test client to catch a multi tenant error.
		azureClient := resources.NewGroupsClient(subscriptionID)
		authorizer, err := credentials.Authorizer()
		if err != nil {
			return nil, microerror.Mask(err)
		}
		azureClient.Client.Authorizer = authorizer
		_, err = azureClient.List(ctx, "", to.Int32Ptr(1))
		if IsApplicationNotFoundInADError(err) {
			k.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("Tried to set up auxiliary tenant %q but authentication failed. Disabling multi-tenant", k.mcTenantID))
			credentials.AuxTenants = oriAuxTenants
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return &AzureClientCredentialsConfig{
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
