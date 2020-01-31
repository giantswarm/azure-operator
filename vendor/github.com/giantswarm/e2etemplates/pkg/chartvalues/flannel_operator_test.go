package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newFlannelOperatorConfigFromFilled(modifyFunc func(*FlannelOperatorConfig)) FlannelOperatorConfig {
	c := FlannelOperatorConfig{
		ClusterName: "test-cluster",
		ClusterRole: FlannelOperatorClusterRole{
			BindingName: "test-flannel-operator",
			Name:        "test-flannel-operator",
		},
		ClusterRolePSP: FlannelOperatorClusterRole{
			BindingName: "test-flannel-operator-psp",
			Name:        "test-flannel-operator-psp",
		},
		Namespace:          "test-namespace",
		RegistryPullSecret: "test-registry-pull-secret",
		PSP: FlannelOperatorPSP{
			Name: "test-psp",
		},
	}

	modifyFunc(&c)
	return c
}

func Test_NewFlannelOperator(t *testing.T) {
	testCases := []struct {
		name           string
		config         FlannelOperatorConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         FlannelOperatorConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newFlannelOperatorConfigFromFilled(func(v *FlannelOperatorConfig) {}),
			expectedValues: `
clusterRoleBindingName: test-flannel-operator
clusterRoleBindingNamePSP: test-flannel-operator-psp
clusterRoleName: test-flannel-operator
clusterRoleNamePSP: test-flannel-operator-psp
Installation:
  V1:
    GiantSwarm:
      FlannelOperator:
        CRD:
          LabelSelector: 'giantswarm.io/cluster=test-cluster'
    Secret:
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
namespace: test-namespace
pspName: test-psp
`,
			errorMatcher: nil,
		},
		{
			name: "case 2: non-default values set",
			config: FlannelOperatorConfig{
				ClusterName: "test-cluster",
				ClusterRole: FlannelOperatorClusterRole{
					BindingName: "test-flannel-operator",
					Name:        "test-flannel-operator",
				},
				ClusterRolePSP: FlannelOperatorClusterRole{
					BindingName: "test-flannel-operator-psp",
					Name:        "test-flannel-operator-psp",
				},
				RegistryPullSecret: "test-registry-pull-secret",
				PSP: FlannelOperatorPSP{
					Name: "test-psp",
				},
			},
			expectedValues: `
clusterRoleBindingName: test-flannel-operator
clusterRoleBindingNamePSP: test-flannel-operator-psp
clusterRoleName: test-flannel-operator
clusterRoleNamePSP: test-flannel-operator-psp
Installation:
  V1:
    GiantSwarm:
      FlannelOperator:
        CRD:
          LabelSelector: 'giantswarm.io/cluster=test-cluster'
    Secret:
      Registry:
        PullSecret:
          DockerConfigJSON: "{\"auths\":{\"quay.io\":{\"auth\":\"test-registry-pull-secret\"}}}"
namespace: giantswarm
pspName: test-psp
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewFlannelOperator(tc.config)

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

func Test_NewFlannelOperator_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       FlannelOperatorConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .ClusterName",
			config: newFlannelOperatorConfigFromFilled(func(v *FlannelOperatorConfig) {
				v.ClusterName = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .ClusterRole.BindingName",
			config: newFlannelOperatorConfigFromFilled(func(v *FlannelOperatorConfig) {
				v.ClusterRole.BindingName = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 2: invalid .ClusterRole.Name",
			config: newFlannelOperatorConfigFromFilled(func(v *FlannelOperatorConfig) {
				v.ClusterRole.Name = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 3: invalid .ClusterRolePSP.BindingName",
			config: newFlannelOperatorConfigFromFilled(func(v *FlannelOperatorConfig) {
				v.ClusterRolePSP.BindingName = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 4: invalid .ClusterRolePSP.Name",
			config: newFlannelOperatorConfigFromFilled(func(v *FlannelOperatorConfig) {
				v.ClusterRolePSP.Name = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 5: invalid .PSP.Name",
			config: newFlannelOperatorConfigFromFilled(func(v *FlannelOperatorConfig) {
				v.PSP.Name = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 6: invalid .RegistryPullSecret",
			config: newFlannelOperatorConfigFromFilled(func(v *FlannelOperatorConfig) {
				v.RegistryPullSecret = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewFlannelOperator(tc.config)

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
