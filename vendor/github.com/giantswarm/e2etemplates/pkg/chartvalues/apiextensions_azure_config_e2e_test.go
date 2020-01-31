package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newAPIExtensionsAzureConfigE2EConfigFromFilled(modifyFunc func(*APIExtensionsAzureConfigE2EConfig)) APIExtensionsAzureConfigE2EConfig {
	c := APIExtensionsAzureConfigE2EConfig{
		Azure: APIExtensionsAzureConfigE2EConfigAzure{
			AvailabilityZones: []int{},
			CalicoSubnetCIDR:  "test-calico-subnet-cidr",
			CIDR:              "test-cidr",
			Location:          "test-location",
			MasterSubnetCIDR:  "test-master-subnet-cidr",
			VMSizeMaster:      "test-vm-size-master",
			VMSizeWorker:      "test-vm-size-worker",
			VPNSubnetCIDR:     "test-vpn-subnet-cidr",
			WorkerSubnetCIDR:  "test-worker-subnet-cidr",
		},
		ClusterName:               "test-cluster-name",
		CommonDomain:              "test-common-domain",
		CommonDomainResourceGroup: "test-common-domain-resource-group",
		SSHPublicKey:              "some-ssh-public-key",
		SSHUser:                   "test-user",
		VersionBundleVersion:      "test-version-bundle-version",
	}

	modifyFunc(&c)

	return c
}

func Test_NewAPIExtensionsAzureConfigE2E(t *testing.T) {
	testCases := []struct {
		name           string
		config         APIExtensionsAzureConfigE2EConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         APIExtensionsAzureConfigE2EConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {}),
			expectedValues: `
azure:
  calicoSubnetCIDR: test-calico-subnet-cidr
  cidr: test-cidr
  location: test-location
  masterSubnetCIDR: test-master-subnet-cidr
  vmSizeMaster: test-vm-size-master
  vmSizeWorker: test-vm-size-worker
  vpnSubnetCIDR: test-vpn-subnet-cidr
  workerSubnetCIDR: test-worker-subnet-cidr
clusterName: test-cluster-name
commonDomain: test-common-domain
commonDomainResourceGroup: test-common-domain-resource-group
sshPublicKey: some-ssh-public-key
sshUser: test-user
versionBundleVersion: test-version-bundle-version
`,
			errorMatcher: nil,
		},
		{
			name: "case 2: all optional values left",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.Azure.VMSizeMaster = ""
				v.Azure.VMSizeWorker = ""
			}),
			expectedValues: `
azure:
  calicoSubnetCIDR: test-calico-subnet-cidr
  cidr: test-cidr
  location: test-location
  masterSubnetCIDR: test-master-subnet-cidr
  vmSizeMaster: Standard_D2s_v3
  vmSizeWorker: Standard_D2s_v3
  vpnSubnetCIDR: test-vpn-subnet-cidr
  workerSubnetCIDR: test-worker-subnet-cidr
clusterName: test-cluster-name
commonDomain: test-common-domain
commonDomainResourceGroup: test-common-domain-resource-group
sshPublicKey: some-ssh-public-key
sshUser: test-user
versionBundleVersion: test-version-bundle-version
`,
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewAPIExtensionsAzureConfigE2E(tc.config)

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

func Test_NewAPIExtensionsAzureConfigE2E_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       APIExtensionsAzureConfigE2EConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .Azure.CalicoSubnetCIDR",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.Azure.CalicoSubnetCIDR = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 1: invalid .Azure.CIDR",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.Azure.CIDR = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 2: invalid .Azure.Location",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.Azure.Location = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 3: invalid .Azure.MasterSubnetCIDR",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.Azure.MasterSubnetCIDR = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 4: invalid .Azure.VPNSubnetCIDR",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.Azure.VPNSubnetCIDR = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 5: invalid .Azure.WorkerSubnetCIDR",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.Azure.WorkerSubnetCIDR = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 6: invalid .ClusterName",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.ClusterName = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 7: invalid .CommonDomain",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.CommonDomain = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 8: invalid .CommonDomainResourceGroup",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.CommonDomainResourceGroup = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 9: invalid .SSHPublicKey",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.SSHPublicKey = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 10: invalid .SSHUser",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.SSHUser = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
		{
			name: "case 11: invalid .VersionBundleVersion",
			config: newAPIExtensionsAzureConfigE2EConfigFromFilled(func(v *APIExtensionsAzureConfigE2EConfig) {
				v.VersionBundleVersion = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAPIExtensionsAzureConfigE2E(tc.config)

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
