package credential

import (
	"context"
	"fmt"
	"os"
	"testing"

	apiextensionslabels "github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/unittest"
)

const (
	clientIDFromCredentialSecret     = "clientIDFromCredentialSecret"
	clientSecretFromCredentialSecret = "clientSecretFromCredentialSecret"
)

func TestParametersOverwriteCredentialsFromEnvironment(t *testing.T) {
	expectedClientID := "parameterClientID"
	expectedClientSecret := "parameterClientSecret"
	expectedTenantID := "parameterTenantID"

	err := os.Setenv("AZURE_CLIENT_ID", "nonExpectedClientID")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Setenv("AZURE_CLIENT_SECRET", "nonExpectedClientSecret")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Setenv("AZURE_TENANT_ID", "nonExpectedTenantID")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = os.Unsetenv("AZURE_CLIENT_ID")
		if err != nil {
			t.Fatal(err)
		}
		err = os.Unsetenv("AZURE_CLIENT_SECRET")
		if err != nil {
			t.Fatal(err)
		}
		err = os.Unsetenv("AZURE_TENANT_ID")
		if err != nil {
			t.Fatal(err)
		}
	}()

	clientCredentialsConfig, err := NewAzureCredentials(expectedClientID, expectedClientSecret, expectedTenantID)
	if err != nil {
		t.Fatal(err)
	}

	if clientCredentialsConfig.ClientID != expectedClientID {
		t.Fatalf("clientID has the wrong value: expected %#q, got %#q", expectedClientID, clientCredentialsConfig.ClientID)
	}
	if clientCredentialsConfig.ClientSecret != expectedClientSecret {
		t.Fatalf("clientSecret has the wrong value: expected %#q, got %#q", expectedClientSecret, clientCredentialsConfig.ClientSecret)
	}
	if clientCredentialsConfig.TenantID != expectedTenantID {
		t.Fatalf("tenantID has the wrong value: expected %#q, got %#q", expectedTenantID, clientCredentialsConfig.TenantID)
	}
}

func TestCredentialsFallbackToEnvVarsWhenMissingParameters(t *testing.T) {
	expectedClientID := "parameterClientID"
	expectedClientSecret := "parameterClientSecret"
	expectedTenantID := "parameterTenantID"

	err := os.Setenv("AZURE_CLIENT_ID", expectedClientID)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Setenv("AZURE_CLIENT_SECRET", expectedClientSecret)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Setenv("AZURE_TENANT_ID", expectedTenantID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = os.Unsetenv("AZURE_CLIENT_ID")
		if err != nil {
			t.Fatal(err)
		}
		err = os.Unsetenv("AZURE_CLIENT_SECRET")
		if err != nil {
			t.Fatal(err)
		}
		err = os.Unsetenv("AZURE_TENANT_ID")
		if err != nil {
			t.Fatal(err)
		}
	}()

	clientCredentialsConfig, err := NewAzureCredentials("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if clientCredentialsConfig.ClientID != expectedClientID {
		t.Fatalf("clientID has the wrong value: expected %#q, got %#q", expectedClientID, clientCredentialsConfig.ClientID)
	}
	if clientCredentialsConfig.ClientSecret != expectedClientSecret {
		t.Fatalf("clientSecret has the wrong value: expected %#q, got %#q", expectedClientSecret, clientCredentialsConfig.ClientSecret)
	}
	if clientCredentialsConfig.TenantID != expectedTenantID {
		t.Fatalf("tenantID has the wrong value: expected %#q, got %#q", expectedTenantID, clientCredentialsConfig.TenantID)
	}
}

func TestNewCredentialsFailWhenMissingConfiguration(t *testing.T) {
	err := os.Setenv("AZURE_CLIENT_ID", "clientID")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Setenv("AZURE_CLIENT_SECRET", "clientSecret")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err = os.Unsetenv("AZURE_CLIENT_ID")
		if err != nil {
			t.Fatal(err)
		}
		err = os.Unsetenv("AZURE_CLIENT_SECRET")
		if err != nil {
			t.Fatal(err)
		}
	}()

	_, err = NewAzureCredentials("", "", "")
	if err == nil {
		t.Fatalf("expected %#q, TenantID is missing", invalidConfigError)
	}
}

func TestCredentialsAreConfiguredUsingOrganizationSecretWithSingleTenantServicePrincipal(t *testing.T) {
	fakeK8sClient := unittest.FakeK8sClient()
	ctx := context.Background()
	tenantID := "sameTenantIDForAuthenticationAndManagingAzureResources"
	expectedClientID := clientIDFromCredentialSecret
	expectedClientSecret := clientSecretFromCredentialSecret
	expectedTenantID := tenantID

	azureCluster, err := setupAuthCRs(ctx, fakeK8sClient, expectedClientID, expectedClientSecret, expectedTenantID, "random-subscription-id", "")
	if err != nil {
		t.Fatal(err)
	}

	credentialProvider := NewK8SCredentialProvider(fakeK8sClient, microloggertest.New())
	clientCredentialsConfig, _, _, err := credentialProvider.GetOrganizationAzureCredentials(ctx, azureCluster.Name)
	if err != nil {
		t.Fatal(err)
	}

	if clientCredentialsConfig.ClientID != expectedClientID {
		t.Fatalf("clientID has the wrong value: expected %#q, got %#q", expectedClientID, clientCredentialsConfig.ClientID)
	}
	if clientCredentialsConfig.ClientSecret != expectedClientSecret {
		t.Fatalf("clientSecret has the wrong value: expected %#q, got %#q", expectedClientSecret, clientCredentialsConfig.ClientSecret)
	}
	if clientCredentialsConfig.TenantID != expectedTenantID {
		t.Fatalf("tenantID has the wrong value: expected %#q, got %#q", expectedTenantID, clientCredentialsConfig.TenantID)
	}
	if len(clientCredentialsConfig.AuxTenants) > 0 {
		t.Fatalf("there shouldn't be any auxiliary tenants for a single tenant service principal: expected 0, got %#q auxiliary tenants", clientCredentialsConfig.AuxTenants)
	}
}

func TestCredentialsAreConfiguredUsingOrganizationSecretWithMultiTenantServicePrincipal(t *testing.T) {
	fakeK8sClient := unittest.FakeK8sClient()
	ctx := context.Background()
	clientID := clientIDFromCredentialSecret
	clientSecret := clientSecretFromCredentialSecret
	servicePrincipalTenantID := "servicePrincipalTenantID"
	subscriptionID := "subscriptionID"
	subscriptionTenantID := "subscriptionTenantID"

	azureCluster, err := setupAuthCRs(ctx, fakeK8sClient, clientID, clientSecret, servicePrincipalTenantID, subscriptionID, subscriptionTenantID)
	if err != nil {
		t.Fatal(err)
	}

	credentialProvider := NewK8SCredentialProvider(fakeK8sClient, microloggertest.New())
	clientCredentialsConfig, _, _, err := credentialProvider.GetOrganizationAzureCredentials(ctx, azureCluster.Name)
	if err != nil {
		t.Fatal(err)
	}

	if clientCredentialsConfig.ClientID != clientID {
		t.Fatalf("clientID has the wrong value: expected %#q, got %#q", clientID, clientCredentialsConfig.ClientID)
	}
	if clientCredentialsConfig.ClientSecret != clientSecret {
		t.Fatalf("clientSecret has the wrong value: expected %#q, got %#q", clientSecret, clientCredentialsConfig.ClientSecret)
	}
	if clientCredentialsConfig.TenantID != subscriptionTenantID {
		t.Fatalf("tenantID has the wrong value: expected %#q, got %#q", subscriptionTenantID, clientCredentialsConfig.TenantID)
	}
	if len(clientCredentialsConfig.AuxTenants) != 1 {
		t.Fatalf("giantswarm tenant id should be an auxiliary tenant id for a multi tenant service principal: expected 1, got %#q auxiliary tenants", clientCredentialsConfig.AuxTenants)
	}

	if clientCredentialsConfig.AuxTenants[0] != servicePrincipalTenantID {
		t.Fatalf("the auxiliary tenant id should have been %#q, got %#q", servicePrincipalTenantID, clientCredentialsConfig.AuxTenants)
	}
}

func TestWhenAzureClusterDoesNotExist(t *testing.T) {
	fakeK8sClient := unittest.FakeK8sClient()
	ctx := context.Background()

	credentialProvider := NewK8SCredentialProvider(fakeK8sClient, microloggertest.New())
	_, _, _, err := credentialProvider.GetOrganizationAzureCredentials(ctx, "ab123")
	if err == nil {
		t.Fatalf("it should fail when organization credential secret is missing")
	}
}

func TestFailsToCreateCredentialsWhenSubscriptionIDIsNotSet(t *testing.T) {
	fakeK8sClient := unittest.FakeK8sClient()
	ctx := context.Background()

	// Missing subscription id
	azureCluster, err := setupAuthCRs(ctx, fakeK8sClient, "client", "secret", "tenant", "", "")
	if err != nil {
		t.Fatal(err)
	}

	credentialProvider := NewK8SCredentialProvider(fakeK8sClient, microloggertest.New())
	_, _, _, err = credentialProvider.GetOrganizationAzureCredentials(ctx, azureCluster.Name)
	if err == nil {
		t.Fatalf("it should fail when key is missing from credential secret: client id is missing")
	}
}

func TestFailsToCreateCredentialsWhenIdentityRefIsNotSet(t *testing.T) {
	fakeK8sClient := unittest.FakeK8sClient()
	ctx := context.Background()

	azureCluster, err := setupAuthCRs(ctx, fakeK8sClient, "client", "secret", "tenant", "subscription", "")
	if err != nil {
		t.Fatal(err)
	}

	// Remove identityRef
	azureCluster.Spec.IdentityRef = nil
	err = fakeK8sClient.CtrlClient().Update(ctx, azureCluster)
	if err != nil {
		t.Fatal(err)
	}

	credentialProvider := NewK8SCredentialProvider(fakeK8sClient, microloggertest.New())
	_, _, _, err = credentialProvider.GetOrganizationAzureCredentials(ctx, azureCluster.Name)
	if err == nil {
		t.Fatalf("it should fail when key is missing from credential secret: client id is missing")
	}
}

func createAzureCluster(k8sclient k8sclient.Interface, ctx context.Context, clusterID, organization string, azureClusterIdentity v1alpha3.AzureClusterIdentity, subscriptionID, subscriptionTenantID string) (*v1alpha3.AzureCluster, error) {
	azureCluster := &v1alpha3.AzureCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterID,
			Namespace: fmt.Sprintf("org-%s", organization),
			Labels: map[string]string{
				"giantswarm.io/cluster": clusterID,
			},
			Annotations: map[string]string{
				SubscriptionTenantIDAnnotation: subscriptionTenantID,
			},
		},
		Spec: v1alpha3.AzureClusterSpec{
			SubscriptionID: subscriptionID,
			IdentityRef: &v1.ObjectReference{
				Kind:      azureClusterIdentity.Kind,
				Namespace: azureClusterIdentity.Namespace,
				Name:      azureClusterIdentity.Name,
			},
		},
	}

	err := k8sclient.CtrlClient().Create(ctx, azureCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureCluster, nil
}

func createAzureClusterIdentity(k8sclient k8sclient.Interface, ctx context.Context, secret v1.Secret, clientID, tenantID string) (*v1alpha3.AzureClusterIdentity, error) {
	azureClusterIdentity := &v1alpha3.AzureClusterIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Labels:    secret.Labels,
		},
		Spec: v1alpha3.AzureClusterIdentitySpec{
			ClientID: clientID,
			ClientSecret: v1.SecretReference{
				Name:      secret.Name,
				Namespace: secret.Namespace,
			},
			TenantID:          tenantID,
			AllowedNamespaces: []string{},
		},
	}
	err := k8sclient.CtrlClient().Create(ctx, azureClusterIdentity)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClusterIdentity, nil
}

func createSecret(k8sclient k8sclient.Interface, ctx context.Context, name string, labels map[string]string, data map[string][]byte) (*v1.Secret, error) {
	organizationCredentialSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "giantswarm",
			Labels:    labels,
		},
		Data: data,
	}
	err := k8sclient.CtrlClient().Create(ctx, organizationCredentialSecret)
	if err != nil {
		return &v1.Secret{}, microerror.Mask(err)
	}

	return organizationCredentialSecret, nil
}

func setupAuthCRs(ctx context.Context, fakeK8sClient k8sclient.Interface, clientID, clientSecret, spTenantID, subscriptionID, subscriptionTenantID string) (*v1alpha3.AzureCluster, error) {
	// Create Secret.
	organization := "giantswarm"
	clusterID := "ab123"
	secret, err := createSecret(
		fakeK8sClient,
		ctx,
		fmt.Sprintf("org-credentials-%s", organization),
		map[string]string{
			apiextensionslabels.ManagedBy:    project.Name(),
			apiextensionslabels.Organization: organization,
		},
		map[string][]byte{
			clientSecretKey: []byte(clientSecret),
		},
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Create AzureClusterIdentity
	azureClusterIdentity, err := createAzureClusterIdentity(fakeK8sClient, ctx, *secret, clientID, spTenantID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	azureCluster, err := createAzureCluster(fakeK8sClient, ctx, clusterID, organization, *azureClusterIdentity, subscriptionID, subscriptionTenantID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureCluster, nil
}
