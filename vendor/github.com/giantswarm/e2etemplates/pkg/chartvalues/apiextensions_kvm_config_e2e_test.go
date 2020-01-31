package chartvalues

import "testing"

func newAPIExtensionsKVMConfigE2EConfigFromFilled(modifyFunc func(*APIExtensionsKVMConfigE2EConfig)) APIExtensionsKVMConfigE2EConfig {
	c := APIExtensionsKVMConfigE2EConfig{
		ClusterID:            "abcde",
		HttpNodePort:         80,
		HttpsNodePort:        443,
		VersionBundleVersion: "2.6.0",
		VNI:                  2,
	}

	modifyFunc(&c)

	return c
}

func Test_NewAPIExtensionsKVMConfigE2E(t *testing.T) {
	testCases := []struct {
		name           string
		config         APIExtensionsKVMConfigE2EConfig
		expectedValues string
		errorMatcher   func(error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         APIExtensionsKVMConfigE2EConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newAPIExtensionsKVMConfigE2EConfigFromFilled(func(v *APIExtensionsKVMConfigE2EConfig) {}),
			expectedValues: `
baseDomain: "k8s.gastropod.gridscale.kvm.gigantic.io"
cluster:
  id: "abcde"
encryptionKey: "QitRZGlWeW5WOFo2YmdvMVRwQUQ2UWoxRHZSVEF4MmovajlFb05sT1AzOD0="
kvm:
  vni: 2 
  ingress:
    httpNodePort: 80 
    httpTargetPort: 30010
    httpsNodePort: 443 
    httpsTargetPort: 30011
sshUser: "test-user"
sshPublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAYQCurvzg5Ia54kb3NZapA6yP00//+Jt6XJNeC7Seq3TeCqMR9x7Snalj19r0lWok1PkRgDo1PXj+3y53zo/wqBrPqN4cQqp00R06kNfnhAgesaRMvYhuyVRQQbfXV5gQg8M= dummy-key"
versionBundle:
  version: "2.6.0"
`,
			errorMatcher: nil,
		},
		{
			name:           "case 2: empty ClusterID",
			config:         newAPIExtensionsKVMConfigE2EConfigFromFilled(func(v *APIExtensionsKVMConfigE2EConfig) { v.ClusterID = "" }),
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:           "case 3: negative HttpNodePort",
			config:         newAPIExtensionsKVMConfigE2EConfigFromFilled(func(v *APIExtensionsKVMConfigE2EConfig) { v.HttpNodePort = -1 }),
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:           "case 4: negative HttpsNodePort",
			config:         newAPIExtensionsKVMConfigE2EConfigFromFilled(func(v *APIExtensionsKVMConfigE2EConfig) { v.HttpsNodePort = -1 }),
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:           "case 4: empty VersionBundleVersion",
			config:         newAPIExtensionsKVMConfigE2EConfigFromFilled(func(v *APIExtensionsKVMConfigE2EConfig) { v.VersionBundleVersion = "" }),
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:           "case 5: negative VNI",
			config:         newAPIExtensionsKVMConfigE2EConfigFromFilled(func(v *APIExtensionsKVMConfigE2EConfig) { v.VNI = -1 }),
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAPIExtensionsKVMConfigE2E(tc.config)

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
