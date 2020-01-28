// +build k8srequired

package clusterdeletion

import (
	"testing"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/setup"
)

var (
	config        setup.Config
	deletecluster *ClusterDeletion
)

func init() {
	var err error
	{
		config, err = setup.NewConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	var p *Provider
	{
		c := ProviderConfig{
			AzureClient: config.AzureClient,
			G8sClient:   config.Host.G8sClient(),
			Logger:      config.Logger,
			ClusterID:   env.ClusterID(),
		}

		p, err = NewProvider(c)
		if err != nil {
			panic(err.Error())
		}
	}

	{
		c := Config{
			Logger:          config.Logger,
			Provider:        p,
			ClusterID:       env.ClusterID(),
			TargetNamespace: config.Host.TargetNamespace(),
		}

		deletecluster, err = New(c)
		if err != nil {
			panic(err.Error())
		}
	}
}

// TestMain allows us to have common setup and teardown steps that are run
// once for all the tests https://golang.org/pkg/testing/#hdr-Main.
func TestMain(m *testing.M) {
	setup.WrapTestMain(m, config)
}
