package cloudconfig

import (
	"testing"
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
			Fn:   func() error { _, err := renderGetKeyVaultSecretsUnit(); return err },
		},
		{
			Name: "renderEtcdMountUnit",
			Fn:   func() error { _, err := renderEtcdMountUnit(diskParams{}); return err },
		},
		{
			Name: "renderEtcdDiskFormatUnit",
			Fn:   func() error { _, err := renderEtcdDiskFormatUnit(diskParams{}); return err },
		},
		{
			Name: "renderDockerMountUnit",
			Fn:   func() error { _, err := renderDockerMountUnit(diskParams{}); return err },
		},
		{
			Name: "renderDockerDiskFormatUnit",
			Fn:   func() error { _, err := renderDockerDiskFormatUnit(diskParams{}); return err },
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
