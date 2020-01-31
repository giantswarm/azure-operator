package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newNodeOperatorConfigFromFilled(modifyFunc func(*NodeOperatorConfig)) NodeOperatorConfig {
	c := NodeOperatorConfig{
		Namespace:          "test-namespace",
		RegistryPullSecret: "test-registry-pull-secret",
	}

	modifyFunc(&c)
	return c
}

func Test_NewNodeOperator(t *testing.T) {
	testCases := []struct {
		name           string
		config         NodeOperatorConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         NodeOperatorConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newNodeOperatorConfigFromFilled(func(v *NodeOperatorConfig) {}),
			expectedValues: `
Installation:
  V1:
    Secret:
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
namespace: test-namespace
`,
			errorMatcher: nil,
		},
		{
			name: "case 2: non-default values set",
			config: NodeOperatorConfig{
				RegistryPullSecret: "test-registry-pull-secret",
			},
			expectedValues: `
Installation:
  V1:
    Secret:
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
namespace: giantswarm
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewNodeOperator(tc.config)

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

func Test_NewNodeOperator_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       NodeOperatorConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .RegistryPullSecret",
			config: newNodeOperatorConfigFromFilled(func(v *NodeOperatorConfig) {
				v.RegistryPullSecret = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewNodeOperator(tc.config)

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
