package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newCredentialdConfigFromFilled(modifyFunc func(*CredentialdConfig)) CredentialdConfig {
	c := CredentialdConfig{
		AWS: CredentialdConfigAWS{
			CredentialDefault: CredentialdConfigAWSCredentialDefault{
				AdminARN:       "test-aws-credential-default-admin-arn",
				AWSOperatorARN: "test-aws-credential-default-aws-operator-arn",
			},
		},
		Azure: CredentialdConfigAzure{
			CredentialDefault: CredentialdConfigAzureCredentialDefault{
				ClientID:       "test-azure-credential-client-id",
				ClientSecret:   "test-azure-credential-client-secret",
				SubscriptionID: "test-azure-credential-subscription-id",
				TenantID:       "test-azure-credential-tenant-id",
			},
		},
		RegistryPullSecret: "test-registry-pull-secret",
	}

	modifyFunc(&c)

	return c
}

func Test_NewCredentiald(t *testing.T) {
	testCases := []struct {
		name           string
		config         CredentialdConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         CredentialdConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name: "case 1: all values set for AWS",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				v.Azure = CredentialdConfigAzure{}
			}),
			expectedValues: `
deployment:
  replicas: 0
Installation:
  V1:
    Secret:
      Credentiald:
        AWS:
          CredentialDefault:
            AdminARN: "test-aws-credential-default-admin-arn"
            AWSOperatorARN: "test-aws-credential-default-aws-operator-arn"
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
`,
			errorMatcher: nil,
		},
		{
			name: "case 2: all values set for Azure",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				v.AWS = CredentialdConfigAWS{}
			}),
			expectedValues: `
deployment:
  replicas: 0
Installation:
  V1:
    Secret:
      Credentiald:
        Azure:
          CredentialDefault:
            ClientID: "test-azure-credential-client-id"
            ClientSecret: "test-azure-credential-client-secret"
            SubscriptionID: "test-azure-credential-subscription-id"
            TenantID: "test-azure-credential-tenant-id"
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
`,
			errorMatcher: nil,
		},
		{
			name: "case 3: non-default values set",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				// Unset Azure to bypass mutual exclusion validation.
				v.Azure = CredentialdConfigAzure{}

				v.Deployment.Replicas = 1
			}),
			expectedValues: `
deployment:
  replicas: 1
Installation:
  V1:
    Secret:
      Credentiald:
        AWS:
          CredentialDefault:
            AdminARN: "test-aws-credential-default-admin-arn"
            AWSOperatorARN: "test-aws-credential-default-aws-operator-arn"
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
`,
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewCredentiald(tc.config)

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

func Test_NewCredentiald_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       CredentialdConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: .AWS and .Azure set at the same time",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: .AWS and .Azure not set",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				v.AWS = CredentialdConfigAWS{}
				v.Azure = CredentialdConfigAzure{}
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 2: invalid .AWS.CredentialDefault.AdminARN",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				// Unset Azure to bypass mutual exclusion validation.
				v.Azure = CredentialdConfigAzure{}

				v.AWS.CredentialDefault.AdminARN = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 3: invalid .AWS.CredentialDefault.AWSOperatorARN",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				// Unset Azure to bypass mutual exclusion validation.
				v.Azure = CredentialdConfigAzure{}

				v.AWS.CredentialDefault.AWSOperatorARN = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 4: invalid .Azure.CredentialDefault.ClientID",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				// Unset AWS to bypass mutual exclusion validation.
				v.AWS = CredentialdConfigAWS{}

				v.Azure.CredentialDefault.ClientID = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 5: invalid .Azure.CredentialDefault.ClientSecret",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				// Unset AWS to bypass mutual exclusion validation.
				v.AWS = CredentialdConfigAWS{}

				v.Azure.CredentialDefault.ClientSecret = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 6: invalid .Azure.CredentialDefault.SubscriptionID",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				// Unset AWS to bypass mutual exclusion validation.
				v.AWS = CredentialdConfigAWS{}

				v.Azure.CredentialDefault.SubscriptionID = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 7: invalid .Azure.CredentialDefault.TenantID",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				// Unset AWS to bypass mutual exclusion validation.
				v.AWS = CredentialdConfigAWS{}

				v.Azure.CredentialDefault.TenantID = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 8: invalid .RegistryPullSecret",
			config: newCredentialdConfigFromFilled(func(v *CredentialdConfig) {
				// Unset AWS to bypass mutual exclusion validation.
				v.AWS = CredentialdConfigAWS{}

				v.RegistryPullSecret = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCredentiald(tc.config)

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
