package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newAPIExtensionsAWSConfigE2EConfigFromFilled(modifyFunc func(*APIExtensionsAWSConfigE2EConfig)) APIExtensionsAWSConfigE2EConfig {
	c := APIExtensionsAWSConfigE2EConfig{
		CommonDomain:         "test-common-domain",
		ClusterName:          "test-cluster-name",
		VersionBundleVersion: "test-version-bundle-version",

		AWS: APIExtensionsAWSConfigE2EConfigAWS{
			APIHostedZone:     "test-api-hosted-zone",
			IngressHostedZone: "test-ingress-hosted-zone",
			NetworkCIDR:       "test-network-cidr",
			PrivateSubnetCIDR: "test-private-subnet-cidr",
			PublicSubnetCIDR:  "test-public-subnet-cidr",
			Region:            "test-region",
			RouteTable0:       "test-route-table-0",
			RouteTable1:       "test-route-table-1",
		},
	}

	modifyFunc(&c)
	return c
}

func Test_NewAPIExtensionsAWSConfigE2E(t *testing.T) {
	testCases := []struct {
		name           string
		config         APIExtensionsAWSConfigE2EConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         APIExtensionsAWSConfigE2EConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {}),
			expectedValues: `commonDomain: test-common-domain
clusterName: test-cluster-name
clusterVersion: v_0_1_0
versionBundleVersion: test-version-bundle-version
aws:
  networkCIDR: test-network-cidr
  privateSubnetCIDR: test-private-subnet-cidr
  publicSubnetCIDR: test-public-subnet-cidr
  region: test-region
  apiHostedZone: test-api-hosted-zone
  ingressHostedZone: test-ingress-hosted-zone
  routeTable0: test-route-table-0
  routeTable1: test-route-table-1
`,
			errorMatcher: nil,
		},
		{
			name: "case 2: all optional values left",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {
				v.AWS.NetworkCIDR = ""
				v.AWS.PrivateSubnetCIDR = ""
				v.AWS.PublicSubnetCIDR = ""
			}),
			expectedValues: `commonDomain: test-common-domain
clusterName: test-cluster-name
clusterVersion: v_0_1_0
versionBundleVersion: test-version-bundle-version
aws:
  networkCIDR: 10.12.0.0/24
  privateSubnetCIDR: 10.12.0.0/25
  publicSubnetCIDR: 10.12.0.128/25
  region: test-region
  apiHostedZone: test-api-hosted-zone
  ingressHostedZone: test-ingress-hosted-zone
  routeTable0: test-route-table-0
  routeTable1: test-route-table-1
`,
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewAPIExtensionsAWSConfigE2E(tc.config)

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

func Test_NewAPIExtensionsAWSConfigE2E_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       APIExtensionsAWSConfigE2EConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .CommonDomain",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {
				v.CommonDomain = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .ClusterName",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {
				v.ClusterName = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 2: invalid .VersionBundleVersion",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {
				v.VersionBundleVersion = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 3: invalid .AWS.APIHostedZone",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {
				v.AWS.APIHostedZone = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 4: invalid .AWS.IngressHostedZone",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {
				v.AWS.IngressHostedZone = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 5: invalid .AWS.Region",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {
				v.AWS.Region = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 6: invalid .AWS.RouteTable0",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {
				v.AWS.RouteTable0 = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 7: invalid .AWS.RouteTable1",
			config: newAPIExtensionsAWSConfigE2EConfigFromFilled(func(v *APIExtensionsAWSConfigE2EConfig) {
				v.AWS.RouteTable1 = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAPIExtensionsAWSConfigE2E(tc.config)

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
