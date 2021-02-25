package credentialprovider

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

type CLIFlagsCredentialProviderConfig struct {
	CtrlClient ctrl.Client
	Logger     micrologger.Logger

	ManagementClusterClientID       string
	ManagementClusterClientSecret   string
	ManagementClusterSubscriptionID string
	ManagementClusterTenantID       string
}

type CLIFlagsCredentialProvider struct {
	ctrlClient ctrl.Client
	logger     micrologger.Logger

	managementClusterClientID       string
	managementClusterClientSecret   string
	managementClusterSubscriptionID string
	managementClusterTenantID       string
}

func NewCLIFlagsCredentialProvider(config CLIFlagsCredentialProviderConfig) (*CLIFlagsCredentialProvider, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.ManagementClusterClientID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ManagementClusterClientID must not be empty", config)
	}
	if config.ManagementClusterClientSecret == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ManagementClusterClientSecret must not be empty", config)
	}
	if config.ManagementClusterSubscriptionID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ManagementClusterSubscriptionID must not be empty", config)
	}
	if config.ManagementClusterTenantID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ManagementClusterTenantID must not be empty", config)
	}

	return &CLIFlagsCredentialProvider{
		ctrlClient:                      config.CtrlClient,
		logger:                          config.Logger,
		managementClusterClientID:       config.ManagementClusterClientID,
		managementClusterClientSecret:   config.ManagementClusterClientSecret,
		managementClusterSubscriptionID: config.ManagementClusterSubscriptionID,
		managementClusterTenantID:       config.ManagementClusterTenantID,
	}, nil
}

func (c *CLIFlagsCredentialProvider) GetLegacyCredentialSecret(ctx context.Context, clusterID string) (*v1alpha1.CredentialSecret, error) {
	return nil, microerror.Maskf(notImplementedError, "GetLegacyCredentialSecret is not implemented for CLIFlagsCredentialProvider")
}

func (c *CLIFlagsCredentialProvider) GetAzureClientCredentialsConfig(ctx context.Context, clusterID string) (*AzureClientCredentialsConfig, error) {
	var subscriptionID string
	var auxTenantID string
	{
		if clusterID != "" {
			azureClusters := &v1alpha3.AzureClusterList{}
			err := c.ctrlClient.List(ctx, azureClusters, ctrl.MatchingLabels{label.Cluster: clusterID})
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
			err = c.ctrlClient.Get(ctx, ctrl.ObjectKey{Namespace: azureCluster.Spec.IdentityRef.Namespace, Name: azureCluster.Spec.IdentityRef.Name}, azureClusterIdentity)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			auxTenantID = azureClusterIdentity.Spec.TenantID
			subscriptionID = azureCluster.Spec.SubscriptionID
		}
	}

	credentials := auth.NewClientCredentialsConfig(c.managementClusterClientID, c.managementClusterClientSecret, c.managementClusterTenantID)
	if auxTenantID != "" && auxTenantID != c.managementClusterTenantID {
		auxTenants := []string{auxTenantID}
		credentials.AuxTenants = auxTenants

		// Test client to catch a multi tenant error.
		azureClient := resources.NewGroupsClient(subscriptionID)
		authorizer, err := credentials.Authorizer()
		if err != nil {
			return nil, microerror.Mask(err)
		}
		azureClient.Client.Authorizer = authorizer
		_, err = azureClient.List(ctx, "", to.Int32Ptr(1))
		if IsApplicationNotFoundInADError(err) {
			c.logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("Tried to set up auxiliary tenant %q but authentication failed. Disabling multi-tenant", auxTenantID))
			credentials.AuxTenants = []string{}
		} else if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return &AzureClientCredentialsConfig{
		ClientCredentialsConfig: credentials,
		SubscriptionID:          c.managementClusterSubscriptionID,
	}, nil
}
