package chartvalues

import (
	"testing"

	"github.com/giantswarm/e2etemplates/internal/rendertest"
)

func newE2ESetupVaultConfigFromFilled(modifyFunc func(*E2ESetupVaultConfig)) E2ESetupVaultConfig {
	c := E2ESetupVaultConfig{
		Vault: E2ESetupVaultConfigVault{
			Token: "test-token",
		},
	}

	modifyFunc(&c)

	return c
}

func Test_NewE2ESetupVault(t *testing.T) {
	testCases := []struct {
		name           string
		config         E2ESetupVaultConfig
		expectedValues string
		errorMatcher   func(err error) bool
	}{
		{
			name:           "case 0: invalid config",
			config:         E2ESetupVaultConfig{},
			expectedValues: ``,
			errorMatcher:   IsInvalidConfig,
		},
		{
			name:   "case 1: all values set",
			config: newE2ESetupVaultConfigFromFilled(func(v *E2ESetupVaultConfig) {}),
			expectedValues: `
vault:
  token: test-token
`,
			errorMatcher: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := NewE2ESetupVault(tc.config)

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

func Test_NewE2ESetupVault_invalidConfigError(t *testing.T) {
	testCases := []struct {
		name         string
		config       E2ESetupVaultConfig
		errorMatcher func(err error) bool
	}{
		{
			name: "case 0: invalid .Vault.Token",
			config: newE2ESetupVaultConfigFromFilled(func(v *E2ESetupVaultConfig) {
				v.Vault.Token = ""
			}),
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewE2ESetupVault(tc.config)

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
