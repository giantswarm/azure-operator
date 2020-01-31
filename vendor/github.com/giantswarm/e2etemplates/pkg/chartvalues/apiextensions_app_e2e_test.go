package chartvalues

import (
	"fmt"
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

const (
	expectedValues = `
apps:
  - name: "test-app"
    namespace: "default"
    catalog: "test-app-catalog"
    config:
      configMap:
        name: "test-app-values"
        namespace: "default"
      secret:
        name: "test-app-secrets"
        namespace: "default"
    kubeConfig:
      inCluster: false
      secret:
        name: "test-kubeconfig-secret"
        namespace: "default"
    version: "1.0.0"

appCatalogs:
  - name: "test-app-catalog"
    title: "test-app-catalog"
    description: "giantswarm app catalog"
    logoURL: "http://giantswarm.logo.catalog.png"
    storage:
      type: "helm"
      url: "https://giantswarm.github.com/sample-catalog"

appOperator:
  version: "1.0.0"

configMaps:
  test-app-values:
    test: "values"

namespace: "default"

secrets:
  test-app-secrets:
    test: "secret"`

	expectedValuesWithoutConfig = `
apps:
  - name: "test-app"
    namespace: "default"
    catalog: "test-app-catalog"
    kubeConfig:
      inCluster: false
      secret:
        name: "test-kubeconfig-secret"
        namespace: "default"
    version: "1.0.0"

appCatalogs:
  - name: "test-app-catalog"
    title: "test-app-catalog"
    description: "giantswarm app catalog"
    logoURL: "http://giantswarm.logo.catalog.png"
    storage:
      type: "helm"
      url: "https://giantswarm.github.com/sample-catalog"

appOperator:
  version: "1.0.0"



namespace: "default"

`
)

func newAPIExtensionsAppE2EConfigFromFilled(modifyFunc func(*APIExtensionsAppE2EConfig)) APIExtensionsAppE2EConfig {
	c := APIExtensionsAppE2EConfig{
		App: APIExtensionsAppE2EConfigApp{
			Name:      "test-app",
			Namespace: "default",
			Catalog:   "test-app-catalog",
			Config: APIExtensionsAppE2EConfigAppConfig{
				ConfigMap: APIExtensionsAppE2EConfigAppConfigConfigMap{
					Name:      "test-app-values",
					Namespace: "default",
				},
				Secret: APIExtensionsAppE2EConfigAppConfigSecret{
					Name:      "test-app-secrets",
					Namespace: "default",
				},
			},
			KubeConfig: APIExtensionsAppE2EConfigAppKubeConfig{
				InCluster: false,
				Secret: APIExtensionsAppE2EConfigAppConfigKubeConfigSecret{
					Name:      "test-kubeconfig-secret",
					Namespace: "default",
				},
			},
			Version: "1.0.0",
		},
		AppCatalog: APIExtensionsAppE2EConfigAppCatalog{
			Name:        "test-app-catalog",
			Title:       "test-app-catalog",
			Description: "giantswarm app catalog",
			LogoURL:     "http://giantswarm.logo.catalog.png",
			Storage: APIExtensionsAppE2EConfigAppCatalogStorage{
				Type: "helm",
				URL:  "https://giantswarm.github.com/sample-catalog",
			},
		},
		AppOperator: APIExtensionsAppE2EConfigAppOperator{
			Version: "1.0.0",
		},
		ConfigMap: APIExtensionsAppE2EConfigConfigMap{
			ValuesYAML: `test: "values"`,
		},
		Namespace: "default",
		Secret: APIExtensionsAppE2EConfigSecret{
			ValuesYAML: `test: "secret"`,
		},
	}

	modifyFunc(&c)

	return c
}

func removeConfigMapAndSecret(v *APIExtensionsAppE2EConfig) {
	v.App.Config = APIExtensionsAppE2EConfigAppConfig{}
}

func Test_NewAPIExtensionsAppE2E(t *testing.T) {
	testCases := []struct {
		name           string
		config         APIExtensionsAppE2EConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         APIExtensionsAppE2EConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:           "case 1: all values set",
			config:         newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {}),
			expectedValues: expectedValues,
			errorMatcher:   nil,
		},
		{
			name:           "case 2: no configmap and secret",
			config:         newAPIExtensionsAppE2EConfigFromFilled(removeConfigMapAndSecret),
			expectedValues: expectedValuesWithoutConfig,
			errorMatcher:   nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewAPIExtensionsAppE2E(tc.config)

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

			fmt.Println(values)
			line, difference := rendertest.Diff(values, tc.expectedValues)
			if line > 0 {
				t.Fatalf("line == %d, want 0, diff: %s", line, difference)
			}
		})
	}
}

func Test_NewAPIExtensionsAppE2E_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       APIExtensionsAppE2EConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .App.Catalog",
			config: newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {
				v.App.Catalog = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .App.Name",
			config: newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {
				v.App.Name = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 2: invalid .App.Namespace",
			config: newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {
				v.App.Namespace = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 3: invalid .App.Version",
			config: newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {
				v.App.Version = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 4: invalid .AppOperator.Version",
			config: newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {
				v.AppOperator.Version = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 5: invalid .Namespace",
			config: newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {
				v.Namespace = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 6: invalid .AppCatalog.Name",
			config: newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {
				v.AppCatalog.Name = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 7: invalid .AppCatalog.Storage.Type",
			config: newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {
				v.AppCatalog.Storage.Type = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 8: invalid .AppCatalog.Storage.URL",
			config: newAPIExtensionsAppE2EConfigFromFilled(func(v *APIExtensionsAppE2EConfig) {
				v.AppCatalog.Storage.URL = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAPIExtensionsAppE2E(tc.config)

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
