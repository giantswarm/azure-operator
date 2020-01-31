package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newAPIExtensionsChartE2EConfigFromFilled(modifyFunc func(*APIExtensionsChartE2EConfig)) APIExtensionsChartE2EConfig {
	c := APIExtensionsChartE2EConfig{
		Chart: APIExtensionsChartE2EConfigChart{
			Name:      "test-app",
			Namespace: "default",
			Config: APIExtensionsChartE2EConfigChartConfig{
				ConfigMap: APIExtensionsChartE2EConfigChartConfigConfigMap{
					Name:      "test-app-values",
					Namespace: "default",
				},
				Secret: APIExtensionsChartE2EConfigChartConfigSecret{
					Name:      "test-app-secrets",
					Namespace: "default",
				},
			},
			TarballURL: "https://giantswarm.github.com/sample-catalog/kubernetes-test-app-chart-0.3.0.tgz",
		},
		ChartOperator: APIExtensionsChartE2EConfigChartOperator{
			Version: "1.0.0",
		},
		ConfigMap: APIExtensionsChartE2EConfigConfigMap{
			ValuesYAML: `test: "values"`,
		},
		Namespace: "default",
		Secret: APIExtensionsChartE2EConfigSecret{
			ValuesYAML: `test: "secret"`,
		},
	}

	modifyFunc(&c)

	return c
}

func Test_NewAPIExtensionsChartE2E(t *testing.T) {
	testCases := []struct {
		name           string
		config         APIExtensionsChartE2EConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         APIExtensionsChartE2EConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newAPIExtensionsChartE2EConfigFromFilled(func(v *APIExtensionsChartE2EConfig) {}),
			expectedValues: `
chart:
  name: "test-app"
  namespace: "default"
  config:
    configMap:
      name: "test-app-values"
      namespace: "default"
    secret:
      name: "test-app-secrets"
      namespace: "default"
  tarballURL: "https://giantswarm.github.com/sample-catalog/kubernetes-test-app-chart-0.3.0.tgz"

chartOperator:
  version: "1.0.0"

configMap:
  values:
    test: "values"

namespace: "default"

secret:
  values:
    test: "secret"`,
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewAPIExtensionsChartE2E(tc.config)

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

func Test_NewAPIExtensionsChartE2E_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       APIExtensionsChartE2EConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .Chart.Name",
			config: newAPIExtensionsChartE2EConfigFromFilled(func(v *APIExtensionsChartE2EConfig) {
				v.Chart.Name = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .Chart.Namespace",
			config: newAPIExtensionsChartE2EConfigFromFilled(func(v *APIExtensionsChartE2EConfig) {
				v.Chart.Namespace = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 2: invalid .Chart.TarballURL",
			config: newAPIExtensionsChartE2EConfigFromFilled(func(v *APIExtensionsChartE2EConfig) {
				v.Chart.TarballURL = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 3: invalid .ChartOperator.Version",
			config: newAPIExtensionsChartE2EConfigFromFilled(func(v *APIExtensionsChartE2EConfig) {
				v.ChartOperator.Version = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 4: invalid .Namespace",
			config: newAPIExtensionsChartE2EConfigFromFilled(func(v *APIExtensionsChartE2EConfig) {
				v.Namespace = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAPIExtensionsChartE2E(tc.config)

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
