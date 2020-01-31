package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newAzureOperatorConfigFromFilled(modifyFunc func(*AzureOperatorConfig)) AzureOperatorConfig {
	c := AzureOperatorConfig{
		Provider: AzureOperatorConfigProvider{
			Azure: AzureOperatorConfigProviderAzure{
				HostClusterCidr: "10.0.0.0/16",
				Location:        "test-location",
			},
		},
		Secret: AzureOperatorConfigSecret{
			AzureOperator: AzureOperatorConfigSecretAzureOperator{
				SecretYaml: AzureOperatorConfigSecretAzureOperatorSecretYaml{
					Service: AzureOperatorConfigSecretAzureOperatorSecretYamlService{
						Azure: AzureOperatorConfigSecretAzureOperatorSecretYamlServiceAzure{
							ClientID:       "test-client-id",
							ClientSecret:   "test-client-secret",
							SubscriptionID: "test-subscription-id",
							TenantID:       "test-tenant-id",
							Template: AzureOperatorConfigSecretAzureOperatorSecretYamlServiceAzureTemplate{
								URI: AzureOperatorConfigSecretAzureOperatorSecretYamlServiceAzureTemplateURI{
									Version: "test-version",
								},
							},
						},
						Tenant: AzureOperatorConfigSecretAzureOperatorSecretYamlServiceTenant{
							Ignition: AzureOperatorConfigSecretAzureOperatorSecretYamlServiceTenantIgnition{
								Debug: AzureOperatorConfigSecretAzureOperatorSecretYamlServiceTenantIgnitionDebug{
									Enabled:    true,
									LogsPrefix: "prefix",
									LogsToken:  "token",
								},
							},
						},
					},
				},
			},
			Registry: AzureOperatorConfigSecretRegistry{
				PullSecret: AzureOperatorConfigSecretRegistryPullSecret{
					DockerConfigJSON: "test-docker-config-json",
				},
			},
		},
	}

	modifyFunc(&c)

	return c
}

func Test_NewAzureOperator(t *testing.T) {
	testCases := []struct {
		name           string
		config         AzureOperatorConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         AzureOperatorConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newAzureOperatorConfigFromFilled(func(v *AzureOperatorConfig) {}),
			expectedValues: `
Installation:
  V1:
    Auth:
      Vault:
        Address: http://vault.default.svc.cluster.local:8200
    Guest:
      IPAM:
        NetworkCIDR: "10.12.0.0/16"
        CIDRMask: 24
        PrivateSubnetMask: 25
        PublicSubnetMask: 25
      Kubernetes:
        API:
          Auth:
            Provider:
              OIDC:
                ClientID: ""
                IssueURL: ""
                UsernameClaim: ""
                GroupsClaim: ""
      SSH:
        SSOPublicKey: 'test'
      Update:
        Enabled: true
    Name: ci-azure-operator
    Provider:
      Azure:
        # TODO rename to EnvironmentName. See https://github.com/giantswarm/giantswarm/issues/4124.
        Cloud: AZUREPUBLICCLOUD
        HostCluster:
          CIDR: "10.0.0.0/16"
          ResourceGroup: "godsmack"
          VirtualNetwork: "godsmack"
          VirtualNetworkGateway: "godsmack-vpn-gateway"
        MSI:
          Enabled: true
        Location: test-location
    Registry:
      Domain: quay.io
    Secret:
      AzureOperator:
        SecretYaml: |
          service:
            azure:
              clientid: test-client-id
              clientsecret: test-client-secret
              subscriptionid: test-subscription-id
              tenantid: test-tenant-id
              template:
                uri:
                  version: test-version
            tenant:
              ignition:
                debug:
                  enabled: true
                  logsprefix: prefix
                  logstoken: token
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-docker-config-json\"}}}"
    Security:
      RestrictAccess:
        Enabled: false
`,
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewAzureOperator(tc.config)

			switch {
			case err == nil && tc.errorMatcher == nil:
				// correct; carry on
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if tc.errorMatcher != nil {
				return
			}

			line, difference := rendertest.Diff(values, tc.expectedValues)
			if line > 0 {
				t.Fatalf("line == %d, want 0, diff: %s", line, difference)
			}
		})
	}
}

func Test_NewAzureOperator_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       AzureOperatorConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .Provider.Azure.Region",
			config: newAzureOperatorConfigFromFilled(func(v *AzureOperatorConfig) {
				v.Provider.Azure.Location = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .Secret.AzureOperator.SecretYaml.Service.Azure.ClientID",
			config: newAzureOperatorConfigFromFilled(func(v *AzureOperatorConfig) {
				v.Secret.AzureOperator.SecretYaml.Service.Azure.ClientID = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 2: invalid .Secret.AzureOperator.SecretYaml.Service.Azure.ClientSecret",
			config: newAzureOperatorConfigFromFilled(func(v *AzureOperatorConfig) {
				v.Secret.AzureOperator.SecretYaml.Service.Azure.ClientSecret = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 3: invalid .Secret.AzureOperator.SecretYaml.Service.Azure.SubscriptionID",
			config: newAzureOperatorConfigFromFilled(func(v *AzureOperatorConfig) {
				v.Secret.AzureOperator.SecretYaml.Service.Azure.SubscriptionID = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 4: invalid .Secret.AzureOperator.SecretYaml.Service.Azure.TenantID",
			config: newAzureOperatorConfigFromFilled(func(v *AzureOperatorConfig) {
				v.Secret.AzureOperator.SecretYaml.Service.Azure.TenantID = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 5: invalid .Secret.AzureOperator.SecretYaml.Service.Azure.Template.URI.Version",
			config: newAzureOperatorConfigFromFilled(func(v *AzureOperatorConfig) {
				v.Secret.AzureOperator.SecretYaml.Service.Azure.Template.URI.Version = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 6: invalid .Secret.Registry.PullSecret.DockerConfigJSON",
			config: newAzureOperatorConfigFromFilled(func(v *AzureOperatorConfig) {
				v.Secret.Registry.PullSecret.DockerConfigJSON = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAzureOperator(tc.config)

			switch {
			case err == nil && tc.errorMatcher == nil:
				// correct; carry on
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}

			if tc.errorMatcher != nil {
				return
			}
		})
	}
}
