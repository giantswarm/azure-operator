package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newCertOperatorConfigFromFilled(modifyFunc func(*CertOperatorConfig)) CertOperatorConfig {
	c := CertOperatorConfig{
		ClusterRole: CertOperatorConfigClusterRole{
			BindingName: "test-cert-operator",
			Name:        "test-cert-operator",
		},
		ClusterRolePSP: CertOperatorConfigClusterRole{
			BindingName: "test-cert-operator-psp",
			Name:        "test-cert-operator-psp",
		},
		CommonDomain: "test-common-domain",
		CRD: CertOperatorConfigCRD{
			LabelSelector: "test-label-selector",
		},
		Namespace:          "test-namespace",
		RegistryPullSecret: "test-registry-pull-secret",
		PSP: CertOperatorPSP{
			Name: "test-cert-operator-psp",
		},
		Vault: CertOperatorVault{
			Token: "test-token",
		},
	}

	modifyFunc(&c)

	return c
}

func Test_NewCertOperator(t *testing.T) {
	testCases := []struct {
		name           string
		config         CertOperatorConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         CertOperatorConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newCertOperatorConfigFromFilled(func(v *CertOperatorConfig) {}),
			expectedValues: `
clusterRoleBindingName: test-cert-operator
clusterRoleBindingNamePSP: test-cert-operator-psp
clusterRoleName: test-cert-operator
clusterRoleNamePSP: test-cert-operator-psp
Installation:
  V1:
    Auth:
      Vault:
        Address: http://vault.default.svc.cluster.local:8200
        CA:
          TTL: 1440h
    GiantSwarm:
      CertOperator:
        CRD:
          LabelSelector: test-label-selector
    Guest:
      Kubernetes:
        API:
          EndpointBase: k8s.test-common-domain
    Secret:
      CertOperator:
        SecretYaml: |
          service:
            vault:
              config:
                token: test-token
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
namespace: test-namespace
pspName: test-cert-operator-psp
`,
			errorMatcher: nil,
		},
		{
			name: "case 2: non-default values set",
			config: CertOperatorConfig{
				CommonDomain:       "test-common-domain",
				RegistryPullSecret: "test-registry-pull-secret",
				Vault: CertOperatorVault{
					Token: "test-token",
				},
			},
			expectedValues: `
clusterRoleBindingName: cert-operator
clusterRoleBindingNamePSP: cert-operator-psp
clusterRoleName: cert-operator
clusterRoleNamePSP: cert-operator-psp
Installation:
  V1:
    Auth:
      Vault:
        Address: http://vault.default.svc.cluster.local:8200
        CA:
          TTL: 1440h
    GiantSwarm:
      CertOperator:
        CRD:
          LabelSelector:
    Guest:
      Kubernetes:
        API:
          EndpointBase: k8s.test-common-domain
    Secret:
      CertOperator:
        SecretYaml: |
          service:
            vault:
              config:
                token: test-token
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
namespace: giantswarm
pspName: cert-operator-psp
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewCertOperator(tc.config)

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

func Test_NewCertOperator_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       CertOperatorConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .CommonDomain",
			config: newCertOperatorConfigFromFilled(func(v *CertOperatorConfig) {
				v.CommonDomain = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .RegistryPullSecret",
			config: newCertOperatorConfigFromFilled(func(v *CertOperatorConfig) {
				v.RegistryPullSecret = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 2: invalid .Vault.Token",
			config: newCertOperatorConfigFromFilled(func(v *CertOperatorConfig) {
				v.Vault.Token = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewCertOperator(tc.config)

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
