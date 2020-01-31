package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newE2ESetupCertsConfigFromFilled(modifyFunc func(*E2ESetupCertsConfig)) E2ESetupCertsConfig {
	c := E2ESetupCertsConfig{
		Cluster: E2ESetupCertsConfigCluster{
			ID: "test-id",
		},
		CommonDomain: "test-common-domain",
	}

	modifyFunc(&c)

	return c
}

func Test_NewE2ESetupCerts(t *testing.T) {
	testCases := []struct {
		name           string
		config         E2ESetupCertsConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         E2ESetupCertsConfig{},
			expectedValues: "",
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newE2ESetupCertsConfigFromFilled(func(v *E2ESetupCertsConfig) {}),
			expectedValues: `
cluster:
  id: test-id
commonDomain: test-common-domain
ipSans:
  - "172.31.0.1"
organizations:
  - "system:masters"
`,
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewE2ESetupCerts(tc.config)

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

func Test_NewE2ESetupCerts_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       E2ESetupCertsConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .Cluster.ID",
			config: newE2ESetupCertsConfigFromFilled(func(v *E2ESetupCertsConfig) {
				v.Cluster.ID = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .CommonDomain",
			config: newE2ESetupCertsConfigFromFilled(func(v *E2ESetupCertsConfig) {
				v.CommonDomain = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewE2ESetupCerts(tc.config)

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
