package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newAPIExtensionsFlannelConfigE2EConfigFromFilled(modifyFunc func(*APIExtensionsFlannelConfigE2EConfig)) APIExtensionsFlannelConfigE2EConfig {
	c := APIExtensionsFlannelConfigE2EConfig{
		ClusterID: "abcde",
		Network:   "10.1.8.0/26",
		VNI:       2,
	}

	modifyFunc(&c)

	return c
}

func Test_NewAPIExtensionsFlannelConfigE2E(t *testing.T) {
	testCases := []struct {
		name           string
		config         APIExtensionsFlannelConfigE2EConfig
		expectedValues string
		errorMatcher   func(error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         APIExtensionsFlannelConfigE2EConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newAPIExtensionsFlannelConfigE2EConfigFromFilled(func(v *APIExtensionsFlannelConfigE2EConfig) {}),
			expectedValues: `
clusterName: "abcde"
versionBundleVersion: "0.2.0"
flannel:
  network: "10.1.8.0/26"
  vni: 2 
`,
			errorMatcher: nil,
		},
		{
			name:           "case 2: negative flannel VNI",
			config:         newAPIExtensionsFlannelConfigE2EConfigFromFilled(func(v *APIExtensionsFlannelConfigE2EConfig) { v.VNI = -2 }),
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:           "case 3: empty ClusterID",
			config:         newAPIExtensionsFlannelConfigE2EConfigFromFilled(func(v *APIExtensionsFlannelConfigE2EConfig) { v.ClusterID = "" }),
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:           "case 4: empty Network",
			config:         newAPIExtensionsFlannelConfigE2EConfigFromFilled(func(v *APIExtensionsFlannelConfigE2EConfig) { v.Network = "" }),
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewAPIExtensionsFlannelConfigE2E(tc.config)

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
