package credential

import (
	"context"
	"os"
	"testing"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/provider/v1alpha1"
	apiextensionslabels "github.com/giantswarm/apiextensions/v2/pkg/label"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger/microloggertest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/unittest"
)

const (
	clientIDFromCredentialSecret     = "clientIDFromCredentialSecret"
	clientSecretFromCredentialSecret = "clientSecretFromCredentialSecret"
)

var (
	noLabels = map[string]string{
		label.App:                        "credentiald",
		apiextensionslabels.Organization: "giantswarm",
	}
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

	_, err := createOrganizationNamespace(fakeK8sClient, ctx)
	if err != nil {
		t.Fatal(err)
	}

	organizationCredentialSecret, err := createOrganizationCredentialSecret(fakeK8sClient, ctx, expectedClientID, expectedClientSecret, expectedTenantID, noLabels)
	if err != nil {
		t.Fatal(err)
	}

	azureConfig := createAzureConfigUsingThisOrganizationCredentialSecret("test-cluster", "org-giantswarm", organizationCredentialSecret)
	credentialProvider := K8SCredential{
		gsTenantID: tenantID,
		k8sclient:  fakeK8sClient,
		logger:     microloggertest.New(),
	}
	clientCredentialsConfig, _, _, err := credentialProvider.GetOrganizationAzureCredentials(ctx, &azureConfig.ObjectMeta)
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
	expectedClientID := clientIDFromCredentialSecret
	expectedClientSecret := clientSecretFromCredentialSecret
	expectedTenantID := "giantswarmTenantID"
	organizationTenantID := "differentTenantID"

	_, err := createOrganizationNamespace(fakeK8sClient, ctx)
	if err != nil {
		t.Fatal(err)
	}

	organizationCredentialSecret, err := createOrganizationCredentialSecret(fakeK8sClient, ctx, expectedClientID, expectedClientSecret, organizationTenantID, noLabels)
	if err != nil {
		t.Fatal(err)
	}

	azureConfig := createAzureConfigUsingThisOrganizationCredentialSecret("test-cluster", "org-giantswarm", organizationCredentialSecret)
	credentialProvider := K8SCredential{
		gsTenantID: "giantswarmTenantID",
		k8sclient:  fakeK8sClient,
		logger:     microloggertest.New(),
	}
	clientCredentialsConfig, _, _, err := credentialProvider.GetOrganizationAzureCredentials(ctx, &azureConfig.ObjectMeta)
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
	if len(clientCredentialsConfig.AuxTenants) != 1 {
		t.Fatalf("giantswarm tenant id should be an auxiliary tenant id for a multi tenant service principal: expected 1, got %#q auxiliary tenants", clientCredentialsConfig.AuxTenants)
	}
}

func TestWhenOrganizationSecretDoesntExistThenCredentialsFailToCreate(t *testing.T) {
	fakeK8sClient := unittest.FakeK8sClient()
	ctx := context.Background()

	_, err := createOrganizationNamespace(fakeK8sClient, ctx)
	if err != nil {
		t.Fatal(err)
	}

	azureConfig := createAzureConfigUsingThisOrganizationCredentialSecret("test-cluster", "giantswarm", &v1.Secret{})
	credentialProvider := K8SCredential{
		gsTenantID: "giantswarmTenantID",
		k8sclient:  fakeK8sClient,
		logger:     microloggertest.New(),
	}
	_, _, _, err = credentialProvider.GetOrganizationAzureCredentials(ctx, &azureConfig.ObjectMeta)
	if err == nil {
		t.Fatalf("it should fail when organization credential secret is missing")
	}
}

func TestFailsToCreateCredentialsUsingOrganizationSecretWhenMissingConfigFromSecret(t *testing.T) {
	fakeK8sClient := unittest.FakeK8sClient()
	ctx := context.Background()

	_, err := createOrganizationNamespace(fakeK8sClient, ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Missing client id
	organizationCredentialSecret, err := createOrganizationCredentialSecretWithMissingClientID(fakeK8sClient, ctx)
	if err != nil {
		t.Fatal(err)
	}

	azureConfig := createAzureConfigUsingThisOrganizationCredentialSecret("test-cluster", "org-giantswarm", organizationCredentialSecret)
	credentialProvider := K8SCredential{
		gsTenantID: "giantswarmTenantID",
		k8sclient:  fakeK8sClient,
		logger:     microloggertest.New(),
	}
	_, _, _, err = credentialProvider.GetOrganizationAzureCredentials(ctx, &azureConfig.ObjectMeta)
	if err == nil {
		t.Fatalf("it should fail when key is missing from credential secret: client id is missing")
	}

	// Missing client secret
	organizationCredentialSecret, err = createOrganizationCredentialSecretWithMissingClientSecret(fakeK8sClient, ctx)
	if err != nil {
		t.Fatal(err)
	}
	azureConfig.Spec.Azure.CredentialSecret.Name = organizationCredentialSecret.GetName()
	_, _, _, err = credentialProvider.GetOrganizationAzureCredentials(ctx, &azureConfig.ObjectMeta)
	if err == nil {
		t.Fatalf("it should fail when key is missing from credential secret: client secret is missing")
	}

	// Missing tenant id
	organizationCredentialSecret, err = createOrganizationCredentialSecretWithMissingTenantID(fakeK8sClient, ctx)
	if err != nil {
		t.Fatal(err)
	}
	azureConfig.Spec.Azure.CredentialSecret.Name = organizationCredentialSecret.GetName()
	_, _, _, err = credentialProvider.GetOrganizationAzureCredentials(ctx, &azureConfig.ObjectMeta)
	if err == nil {
		t.Fatalf("it should fail when key is missing from credential secret: tenant id is missing")
	}

	// Missing subscription
	organizationCredentialSecret, err = createOrganizationCredentialSecretWithMissingSubscriptionID(fakeK8sClient, ctx)
	if err != nil {
		t.Fatal(err)
	}
	azureConfig.Spec.Azure.CredentialSecret.Name = organizationCredentialSecret.GetName()
	_, _, _, err = credentialProvider.GetOrganizationAzureCredentials(ctx, &azureConfig.ObjectMeta)
	if err == nil {
		t.Fatalf("it should fail when key is missing from credential secret: subscription ID is missing")
	}
}

func TestCredentialsAreConfiguredUsingOrganizationSecretWithSingleTenantServicePrincipalWhenSecretIsLabeled(t *testing.T) {
	fakeK8sClient := unittest.FakeK8sClient()
	ctx := context.Background()
	expectedClientID := clientIDFromCredentialSecret
	expectedClientSecret := clientSecretFromCredentialSecret
	expectedTenantID := "TenantIDFromCredentialSecret"

	_, err := createOrganizationNamespace(fakeK8sClient, ctx)
	if err != nil {
		t.Fatal(err)
	}

	labels := map[string]string{
		label.SingleTenantSP:             "true",
		label.App:                        "credentiald",
		apiextensionslabels.Organization: "giantswarm",
	}

	organizationCredentialSecret, err := createOrganizationCredentialSecret(fakeK8sClient, ctx, expectedClientID, expectedClientSecret, expectedTenantID, labels)
	if err != nil {
		t.Fatal(err)
	}

	azureConfig := createAzureConfigUsingThisOrganizationCredentialSecret("test-cluster", "org-giantswarm", organizationCredentialSecret)
	credentialProvider := K8SCredential{
		gsTenantID: "giantswarmTenantID",
		k8sclient:  fakeK8sClient,
		logger:     microloggertest.New(),
	}
	clientCredentialsConfig, _, _, err := credentialProvider.GetOrganizationAzureCredentials(ctx, &azureConfig.ObjectMeta)
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
	if len(clientCredentialsConfig.AuxTenants) != 0 {
		t.Fatalf("there shouldn't be an auxiliary tenant id when secret is labeled as single tenant: expected 0, got %#q auxiliary tenants", clientCredentialsConfig.AuxTenants)
	}
}

func createOrganizationCredentialSecret(k8sclient k8sclient.Interface, ctx context.Context, clientID, clientSecret, tenantID string, labels map[string]string) (*v1.Secret, error) {
	data := map[string][]byte{
		clientIDKey:       []byte(clientID),
		clientSecretKey:   []byte(clientSecret),
		tenantIDKey:       []byte(tenantID),
		subscriptionIDKey: []byte("1a2b3c4d-5e6f-7g8h9i"),
		partnerIDKey:      []byte("9i8h7g-6g5e-4d3c2b1a"),
	}
	organizationCredentialSecret, err := createSecret(k8sclient, ctx, "credential-secret", labels, data)
	if err != nil {
		return &v1.Secret{}, microerror.Mask(err)
	}

	return organizationCredentialSecret, nil
}

func createOrganizationCredentialSecretWithMissingClientID(k8sclient k8sclient.Interface, ctx context.Context) (*v1.Secret, error) {
	data := map[string][]byte{
		clientSecretKey:   []byte("irrelevant"),
		tenantIDKey:       []byte("irrelevant"),
		partnerIDKey:      []byte("irrelevant"),
		subscriptionIDKey: []byte("irrelevant"),
	}
	organizationCredentialSecret, err := createSecret(k8sclient, ctx, "credential-multi-tenant-with-missing-clientid", nil, data)
	if err != nil {
		return &v1.Secret{}, microerror.Mask(err)
	}

	return organizationCredentialSecret, nil
}

func createOrganizationCredentialSecretWithMissingClientSecret(k8sclient k8sclient.Interface, ctx context.Context) (*v1.Secret, error) {
	data := map[string][]byte{
		clientIDKey:       []byte("irrelevant"),
		tenantIDKey:       []byte("irrelevant"),
		partnerIDKey:      []byte("irrelevant"),
		subscriptionIDKey: []byte("irrelevant"),
	}
	organizationCredentialSecret, err := createSecret(k8sclient, ctx, "credential-multi-tenant-with-missing-clientsecret", nil, data)
	if err != nil {
		return &v1.Secret{}, microerror.Mask(err)
	}

	return organizationCredentialSecret, nil
}

func createOrganizationCredentialSecretWithMissingTenantID(k8sclient k8sclient.Interface, ctx context.Context) (*v1.Secret, error) {
	data := map[string][]byte{
		clientIDKey:       []byte("irrelevant"),
		clientSecretKey:   []byte("irrelevant"),
		partnerIDKey:      []byte("irrelevant"),
		subscriptionIDKey: []byte("irrelevant"),
	}
	organizationCredentialSecret, err := createSecret(k8sclient, ctx, "credential-multi-tenant-with-missing-tenantid", nil, data)
	if err != nil {
		return &v1.Secret{}, microerror.Mask(err)
	}

	return organizationCredentialSecret, nil
}

func createOrganizationCredentialSecretWithMissingSubscriptionID(k8sclient k8sclient.Interface, ctx context.Context) (*v1.Secret, error) {
	data := map[string][]byte{
		clientIDKey:     []byte("irrelevant"),
		clientSecretKey: []byte("irrelevant"),
		partnerIDKey:    []byte("irrelevant"),
		tenantIDKey:     []byte("irrelevant"),
	}
	organizationCredentialSecret, err := createSecret(k8sclient, ctx, "credential-multi-tenant-with-missing-subscriptionid", nil, data)
	if err != nil {
		return &v1.Secret{}, microerror.Mask(err)
	}

	return organizationCredentialSecret, nil
}

func createOrganizationNamespace(k8sclient k8sclient.Interface, ctx context.Context) (*v1.Namespace, error) {
	organizationNamespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "org-giantswarm",
		},
		Spec: v1.NamespaceSpec{},
	}
	err := k8sclient.CtrlClient().Create(ctx, organizationNamespace)
	if err != nil {
		return &v1.Namespace{}, microerror.Mask(err)
	}

	return organizationNamespace, nil
}

func createSecret(k8sclient k8sclient.Interface, ctx context.Context, name string, labels map[string]string, data map[string][]byte) (*v1.Secret, error) {
	organizationCredentialSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "org-giantswarm",
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

func createAzureConfigUsingThisOrganizationCredentialSecret(name, namespace string, organizationCredentialSecret *v1.Secret) *v1alpha1.AzureConfig {
	return &v1alpha1.AzureConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				label.App:                        "credentiald",
				apiextensionslabels.Organization: "giantswarm",
			},
		},
		Spec: v1alpha1.AzureConfigSpec{
			Azure: v1alpha1.AzureConfigSpecAzure{
				CredentialSecret: v1alpha1.CredentialSecret{
					Name:      organizationCredentialSecret.GetName(),
					Namespace: organizationCredentialSecret.GetNamespace(),
				},
			},
		},
	}
}
