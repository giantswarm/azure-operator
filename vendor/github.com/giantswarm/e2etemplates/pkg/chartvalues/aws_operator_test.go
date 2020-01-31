package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newAWSOperatorConfigFromFilled(modifyFunc func(*AWSOperatorConfig)) AWSOperatorConfig {
	c := AWSOperatorConfig{
		InstallationName: "ci-aws-operator",
		Provider: AWSOperatorConfigProvider{
			AWS: AWSOperatorConfigProviderAWS{
				Encrypter:       "vault",
				Region:          "eu-central-1",
				RouteTableNames: "foo,bar",
			},
		},
		RegistryPullSecret: "test-registry-pull-secret",
		Secret: AWSOperatorConfigSecret{
			AWSOperator: AWSOperatorConfigSecretAWSOperator{
				SecretYaml: AWSOperatorConfigSecretAWSOperatorSecretYaml{
					Service: AWSOperatorConfigSecretAWSOperatorSecretYamlService{
						AWS: AWSOperatorConfigSecretAWSOperatorSecretYamlServiceAWS{
							AccessKey: AWSOperatorConfigSecretAWSOperatorSecretYamlServiceAWSAccessKey{
								ID:     "test-access-key-id",
								Secret: "test-access-key-secret",
								Token:  "test-access-key-token",
							},
							HostAccessKey: AWSOperatorConfigSecretAWSOperatorSecretYamlServiceAWSAccessKey{
								ID:     "test-host-access-key-id",
								Secret: "test-host-access-key-secret",
								Token:  "test-host-access-key-token",
							},
						},
					},
				},
			},
		},
		SSH: AWSOperatorConfigSSH{
			UserList: "test-user-list",
		},
	}

	modifyFunc(&c)
	return c
}

func Test_NewAWSOperator(t *testing.T) {
	testCases := []struct {
		name           string
		config         AWSOperatorConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         AWSOperatorConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {}),
			expectedValues: `Installation:
  V1:
    Auth:
      Vault:
        Address: http://vault.default.svc.cluster.local:8200
    Guest:
      Calico:
        CIDR: 16
        Subnet: "192.168.0.0"
      Docker:
        CIDR: "172.17.0.1/16"
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
          ClusterIPRange: "172.31.0.0/24"
        Kubelet:
          ImagePullProgressDeadline: 1m
      SSH:
        SSOPublicKey: 'test'
        UserList: 'test-user-list'
      Update:
        Enabled: true
    Name: 'ci-aws-operator'
    Provider:
      AWS:
        AvailabilityZones:
          - eu-central-1a
          - eu-central-1b
          - eu-central-1c
        Region: 'eu-central-1'
        DeleteLoggingBucket: true
        IncludeTags: true
        Route53:
          Enabled: true
        RouteTableNames: 'foo,bar'
        Encrypter: 'vault'
        TrustedAdvisor:
          Enabled: false
    Registry:
      Domain: quay.io
    Secret:
      AWSOperator:
        SecretYaml: |
          service:
            aws:
              accesskey:
                id: 'test-access-key-id'
                secret: 'test-access-key-secret'
                token: 'test-access-key-token'
              hostaccesskey:
                id: 'test-host-access-key-id'
                secret: 'test-host-access-key-secret'
                token: 'test-host-access-key-token'
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
    Security:
      RestrictAccess:
        Enabled: false
        GSAPI: false
        GuestAPI:
          Private: false
          Public: false
`,
			errorMatcher: nil,
		},
		{
			name: "case 2: all optional values left",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.Provider.AWS.Encrypter = ""
				v.Secret.AWSOperator.SecretYaml.Service.AWS.AccessKey.Token = ""
				v.Secret.AWSOperator.SecretYaml.Service.AWS.HostAccessKey.Token = ""
			}),
			expectedValues: `Installation:
  V1:
    Auth:
      Vault:
        Address: http://vault.default.svc.cluster.local:8200
    Guest:
      Calico:
        CIDR: 16
        Subnet: "192.168.0.0"
      Docker:
        CIDR: "172.17.0.1/16"
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
          ClusterIPRange: "172.31.0.0/24"
        Kubelet:
          ImagePullProgressDeadline: 1m
      SSH:
        SSOPublicKey: 'test'
        UserList: 'test-user-list'
      Update:
        Enabled: true
    Name: 'ci-aws-operator'
    Provider:
      AWS:
        AvailabilityZones:
          - eu-central-1a
          - eu-central-1b
          - eu-central-1c
        Region: 'eu-central-1'
        DeleteLoggingBucket: true
        IncludeTags: true
        Route53:
          Enabled: true
        RouteTableNames: 'foo,bar'
        Encrypter: 'kms'
        TrustedAdvisor:
          Enabled: false
    Registry:
      Domain: quay.io
    Secret:
      AWSOperator:
        SecretYaml: |
          service:
            aws:
              accesskey:
                id: 'test-access-key-id'
                secret: 'test-access-key-secret'
                token: ''
              hostaccesskey:
                id: 'test-host-access-key-id'
                secret: 'test-host-access-key-secret'
                token: ''
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
    Security:
      RestrictAccess:
        Enabled: false
        GSAPI: false
        GuestAPI:
          Private: false
          Public: false
`,
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewAWSOperator(tc.config)

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

func Test_NewAWSOperator_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       AWSOperatorConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .Provider.AWS.Region",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.Provider.AWS.Region = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .Provider.AWS.RouteTableNames",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.Provider.AWS.RouteTableNames = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 3: invalid .RegistryPullSecret",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.RegistryPullSecret = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 4: invalid .Secret.AWSOperator.SecretYaml.Service.AWS.AccessKey.ID",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.Secret.AWSOperator.SecretYaml.Service.AWS.AccessKey.ID = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 5: invalid .Secret.AWSOperator.SecretYaml.Service.AWS.AccessKey.Secret",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.Secret.AWSOperator.SecretYaml.Service.AWS.AccessKey.Secret = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 6: invalid .Secret.AWSOperator.SecretYaml.Service.AWS.HostAccessKey.ID",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.Secret.AWSOperator.SecretYaml.Service.AWS.HostAccessKey.ID = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 7: invalid .Secret.AWSOperator.SecretYaml.Service.AWS.HostAccessKey.Secret",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.Secret.AWSOperator.SecretYaml.Service.AWS.HostAccessKey.Secret = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 8: invalid .SSH.UserList",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.SSH.UserList = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 9: invalid .InstallationName",
			config: newAWSOperatorConfigFromFilled(func(v *AWSOperatorConfig) {
				v.InstallationName = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAWSOperator(tc.config)

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
