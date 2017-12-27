package cloudconfig

import (
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

func Test_render(t *testing.T) {
	testCases := []struct {
		Name string
		Fn   func() error
	}{
		{
			Name: "renderCalicoAzureFile",
			Fn:   func() error { _, err := renderCalicoAzureFile(calicoAzureFileParams{}); return err },
		},
		{
			Name: "renderCloudProviderConfFile",
			Fn:   func() error { _, err := renderCloudProviderConfFile(cloudProviderConfFileParams{}); return err },
		},
		{
			Name: "renderDefaultStorageClassFile",
			Fn:   func() error { _, err := renderDefaultStorageClassFile(); return err },
		},
		{
			Name: "renderGetKeyVaultSecretsFile",
			Fn:   func() error { _, err := renderGetKeyVaultSecretsFile(getKeyVaultSecretsFileParams{}); return err },
		},
		{
			Name: "renderGetKeyVaultSecretsUnit",
			Fn:   func() error { _, err := renderGetKeyVaultSecretsUnit(providerv1alpha1.AzureConfig{}); return err },
		},
	}

	for i, tc := range testCases {
		// Test if *Params struct have all fields needed to evaluate
		// the template.
		err := tc.Fn()
		if err != nil {
			t.Errorf("case %d: %s: expected err = nil, got %v", i, tc.Name, err)
		}
	}
}
